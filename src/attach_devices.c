
#include "controllers.h"
#include "joycon.h"
#include "loop.h"
#include <hidapi/hidapi.h>

#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wchar.h>

/* Scan system for newly connected Joy-Cons */
void scan_joycons(void) {
	struct hid_device_info *devs, *cur_dev;
	jc_side side;

	devs = hid_enumerate(JOYCON_VENDOR, 0x0);
	cur_dev = devs;
	for (; cur_dev; cur_dev = cur_dev->next) {
		if (cur_dev->product_id == JOYCON_PRODUCT_L) {
			side = JC_LEFT;
		} else if (cur_dev->product_id == JOYCON_PRODUCT_R) {
			side = JC_RIGHT;
		} else {
			continue;
		}
		int gidx;
		bool found = false;
		for (gidx = 0; gidx < MAX_JOYCON; gidx++) {
			if (g_joycons[gidx].serial != NULL &&
			    wcscmp(g_joycons[gidx].serial, cur_dev->serial_number)) {
				found = true;
				break;
			}
		}
		if (found) {
			// Already connected
			if (g_joycons[gidx].status == JC_ST_WANT_RECONNECT) {
				// Reconnect
				errno = 0;
				g_joycons[gidx].hidapi_handle = hid_open_path(cur_dev->path);
				if (g_joycons[gidx].hidapi_handle == NULL) {
					if (errno == EPERM) {
						printf("Error: Not running as root, could not connect "
						       "to %ls\n",
						       cur_dev->serial_number);
						exit(1);
					}
					printf("Error: Could not open device serial=%ls\n",
					       cur_dev->serial_number);
					g_joycons[gidx].status = JC_ST_ERROR;
				}
			}
			continue;
		}
		for (gidx = 0; gidx < MAX_JOYCON; gidx++) {
			if (g_joycons[gidx].status == JC_ST_INVALID) {
				break;
			}
		}
		if (gidx == 10) {
			printf("Error: Too many Joy-Cons connected via Bluetooth\n");
			continue;
		}
		memset(&g_joycons[gidx], 0, sizeof(joycon_state));
		joycon_state *jc = &g_joycons[gidx];
		printf("Found JoyCon %c, #%i: %ls %s\n", side == JC_LEFT ? 'L' : 'R',
		       gidx, cur_dev->serial_number, cur_dev->path);
		jc->serial = wcsdup(cur_dev->serial_number);
		errno = 0;
		jc->hidapi_handle = hid_open_path(cur_dev->path);
		if (jc->hidapi_handle == NULL) {
			int errnum = errno;
			if (errnum == EACCES) {
				printf("Error: Permission failure, could not open path=%s "
				       "serial=%ls\n",
				       cur_dev->path, cur_dev->serial_number);
				printf("Exiting, please update udev or run as root\n");
				exit(1);
			}
			printf(
			    "Error: Could not open device path=%s serial=%ls reason=%s\n",
			    cur_dev->path, cur_dev->serial_number, strerror(errnum));
			g_joycons[gidx].status = JC_ST_ERROR;
			continue;
		}

		// Try to find stick calibration data
		jc->status = JC_ST_WAITING_PAIR;
	}
	hid_free_enumeration(devs);
}

/* Assign one or a pair of Joy-Cons to a controller */
static void assign_controller(joycon_state *jc, joycon_state *jc2) {
	int cidx;
	for (cidx = 0; cidx < MAX_OUTCONTROL; cidx++) {
		if (g_controllers[cidx].active == CONTROLLER_STATUS_INACTIVE) {
			break;
		}
	}
	if (cidx == MAX_OUTCONTROL) {
		printf("Error: Reached maximum output controller number\n");
		return;
	}

	controller_state *c = &g_controllers[cidx];
	memset(c, 0, sizeof(*c));
	if (jc2 == NULL) {
		if (jc->side == JC_LEFT) {
			c->jcl = jc;
		} else {
			c->jcr = jc;
		}
		c->mapping = cmap_default_one_joycon;
	} else {
		c->jcl = jc;
		c->jcr = jc2;
		c->mapping = cmap_default_two_joycons;
	}
	c->active = CONTROLLER_STATUS_SETUP;
}

static const uint8_t SL_SR = 0xFF & (JC_BUTTON_R_SR | JC_BUTTON_R_SL);

static void attempt_pairing(joycon_state *jc) {
	if (((jc->buttons[0] & SL_SR) == SL_SR) ||
	    ((jc->buttons[2] & SL_SR) == SL_SR)) {
		// Pair as single
		printf("Pairing Joy-Con %c solo controller (serial=%ls)\n",
		       jc->side == JC_LEFT ? 'L' : 'R', jc->serial);
		jc->status = JC_ST_ACTIVE;
		assign_controller(jc, NULL);
		return;
	}
	if (jc->side == JC_LEFT && jc_getbutton(JC_BUTTON_L_L, jc)) {
		joycon_state *jc2;
		for (int j = 0; j < MAX_JOYCON; j++) {
			jc2 = &g_joycons[j];
			if (jc2->side == JC_RIGHT && jc2->status == JC_ST_WAITING_PAIR &&
			    jc_getbutton(JC_BUTTON_R_R, jc2)) {
				// Pair as double
				printf("Pairing Joy-Cons as double controller (serial=%ls "
				       "serial=%ls)\n",
				       jc->serial, jc2->serial);
				jc->status = JC_ST_ACTIVE;
				jc2->status = JC_ST_ACTIVE;
				assign_controller(jc, jc2);
				return;
			}
		}
	}
}

void controller_pairing_check(void) {
	// Check for new controller pairing
	for (int i = 0; i < MAX_JOYCON; i++) {
		if (g_joycons[i].status == JC_ST_WAITING_PAIR) {
			attempt_pairing(&g_joycons[i]);
		}
	}
}
