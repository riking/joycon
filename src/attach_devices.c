
#include "controllers.h"
#include "joycon.h"
#include <hidapi/hidapi.h>

#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wchar.h>

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
				g_joycons[gidx].handle = hid_open_path(cur_dev->path);
				if (g_joycons[gidx].handle == NULL) {
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
		jc->handle = hid_open_path(cur_dev->path);
		if (jc->handle == NULL) {
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