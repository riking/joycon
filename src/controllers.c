
#include "controllers.h"
#include "joycon.h"
#include "uinput_keys.h"

#include <hidapi/hidapi.h>
#include <linux/input.h>
#include <linux/uinput.h>

#include <errno.h>
#include <fcntl.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

joycon_state g_joycons[MAX_JOYCON];
controller_state g_controllers[MAX_OUTCONTROL];

int cnum(controller_state *c) {
	return 1 + ((ptrdiff_t)(c - &g_controllers[0])) / sizeof(controller_state);
}

void setup_controller(controller_state *c) {
	int fd;

	if (c->fd != 0) {
		printf("ABORT: cannot reinitialize a controller\n");
		exit(1);
	}

	fd = open("/dev/uinput", O_WRONLY | O_NONBLOCK);
	if (fd < 0) {
		printf("Error starting controller %i: Could not open /dev/uinput\n",
		       cnum(c));
		printf("Fix permissions or relaunch as root\n");
		exit(1);
	}

	c->fd = fd;
	int ret;
	ret = ioctl(c->fd, UI_SET_EVBIT, EV_KEY);
	if (ret < 0) {
		printf("Error starting controller %i: uinput error: %s\n", cnum(c),
		       strerror(errno));
		c->status = CONTROLLER_STATUS_TEARDOWN;
		return;
	}
	ret = ioctl(c->fd, UI_SET_EVBIT, EV_ABS);
	if (ret < 0) {
		printf("Error starting controller %i: uinput error: %s\n", cnum(c),
		       strerror(errno));
		c->status = CONTROLLER_STATUS_TEARDOWN;
		return;
	}
	ioctl(c->fd, UI_SET_ABSBIT, ABS_X);
	ioctl(c->fd, UI_SET_ABSBIT, ABS_Y);

	struct input_id uid;
	struct uinput_user_dev uidev;
	memset(&uidev, 0, sizeof(uidev));

	if (c->jcl && c->jcr) {
		snprintf(uidev.name, UINPUT_MAX_NAME_SIZE, "Joy-Con Pair #%i", cnum(c));
	} else if (c->jcl) {
		snprintf(uidev.name, UINPUT_MAX_NAME_SIZE, "Joy-Con L #%i", cnum(c));
	} else {
		snprintf(uidev.name, UINPUT_MAX_NAME_SIZE, "Joy-Con R #%i", cnum(c));
	}
	printf("connecting uinput '%s'...\n", uidev.name);
	uid.bustype = BUS_BLUETOOTH;
	// uid.bustype = BUS_USB;
	uid.vendor = JOYCON_VENDOR;
	uid.product = JOYCON_PRODUCT_L - 5; // uhhhh just make something up
	uid.version = 1;
	uidev.id = uid;

	// note: because of this, need to re-init controller whenever config changes
	for (size_t i = 0; i < c->mapping.length; i++) {
		if (c->mapping.ptr[i].type == CONTROLLER_MAP_AXIS) {
			int axis = c->mapping.ptr[i].axis.uinput_axis;
			if (ioctl(c->fd, UI_SET_ABSBIT, axis) < 0) {
				printf("Error starting controller %i: uinput error: %s\n",
				       cnum(c), strerror(errno));
				c->status = CONTROLLER_STATUS_TEARDOWN;
				return;
			}
			uidev.absmin[axis] = 0;
			uidev.absmax[axis] = 0xFF;
			uidev.absflat[axis] = 3;
		} else if (c->mapping.ptr[i].type == CONTROLLER_MAP_BUTTON) {
			int btn = c->mapping.ptr[i].button.uinput_button;
			if (ioctl(c->fd, UI_SET_KEYBIT, btn) < 0) {
				printf("Error starting controller %i: %ld %d %s: uinput error: "
				       "%s\n",
				       cnum(c), UI_SET_KEYBIT, btn, uinput_key_name(btn),
				       strerror(errno));
				c->status = CONTROLLER_STATUS_TEARDOWN;
				return;
			}
		}
	}

	ret = write(c->fd, &uidev, sizeof(uidev));
	if (ret < 0) {
		printf("Error starting controller %i: uinput error: %s\n", cnum(c),
		       strerror(errno));
		c->status = CONTROLLER_STATUS_TEARDOWN;
		return;
	}
	ret = ioctl(fd, UI_DEV_CREATE);
	if (ret < 0) {
		printf("Error starting controller %i: uinput error: %s\n", cnum(c),
		       strerror(errno));
		c->status = CONTROLLER_STATUS_TEARDOWN;
		return;
	}
	c->status = CONTROLLER_STATUS_ACTIVE;
}

void destroy_controller(controller_state *c) {
	ioctl(c->fd, UI_DEV_DESTROY);
	close(c->fd);
	c->fd = -1;
	c->status = CONTROLLER_STATUS_INACTIVE;
	if (c->jcl && c->jcl->status == JC_ST_ACTIVE) {
		c->jcl->status = JC_ST_WAITING_PAIR;
	}
	if (c->jcr && c->jcr->status == JC_ST_ACTIVE) {
		c->jcr->status = JC_ST_WAITING_PAIR;
	}
	memset(c, 0, sizeof(controller_state));
}

static void dispatch_buttons(controller_state *c, uint8_t *bu_now,
                             uint8_t *bu_prev) {
	struct input_event ev[8 * 3];
	memset(&ev, 0, sizeof(ev));
	int evi = 0;

	for (int b = 0; b < 3; b++) {
		if (bu_now[b] == bu_prev[b]) {
			continue;
		}

		for (int i = 0; i < 8; i++) {
			if ((bu_now[b] & (1 << i)) != (bu_prev[b] & (1 << i))) {
				jc_button_id bid = ((b + 2) << 8) | (1 << i);
				bool found = false;
				size_t q;
				for (q = 0; q < c->mapping.length; q++) {
					if (c->mapping.ptr[q].type == CONTROLLER_MAP_BUTTON &&
					    c->mapping.ptr[q].button.joycon_button == bid) {
						found = true;
						break;
					}
				}
				if (found) {
					ev[evi].type = EV_KEY;
					ev[evi].value = (bu_now[b] & (1 << i)) != 0;
					ev[evi].code = c->mapping.ptr[q].button.uinput_button;
					evi++;
				}
				printf("Controller #%i: Button %s %s\n", cnum(c),
				       jc_button_name(bid),
				       (bu_now[b] & (1 << i)) != 0 ? "pressed" : "released");
			}
		}
	}
	if (evi > 0) {
		if (write(c->fd, ev, sizeof(ev[0]) * evi) < 0) {
			printf("Write error: %s", strerror(errno));
		}
	}

	if (((bu_now[1] & (JC_BUTTON_R_STI & 0xFF)) != 0) &&
	    !(((bu_prev[1] & (JC_BUTTON_R_STI & 0xFF)) != 0))) {
		printf("sending 0x80...\n");
		uint8_t packet[25];
		memset(packet, 0, sizeof(packet));
		packet[0] = 0x10;
		packet[1] = 0x91;
		packet[2] = 0x01;
		errno = 0;
		int ret = hid_write(c->jcr->hidapi_handle, packet, 8);
		if (ret < 9) {
			printf("failed write %d: %s\n", ret, strerror(errno));
		}
	}
}

static void joycon_died(controller_state *c) {
	c->status = CONTROLLER_STATUS_DEADCON;
	uint8_t buttons[3];
	memset(buttons, 0, sizeof(buttons));
	dispatch_buttons(c, buttons, c->prev_button_state);
	memset(c->prev_button_state, 0, sizeof(c->prev_button_state));
	memset(c->prev_lstick_state, 0, sizeof(c->prev_lstick_state));
	memset(c->prev_rstick_state, 0, sizeof(c->prev_rstick_state));
}

static int get_axis_mapping(controller_state *c, jc_side side,
                            bool is_vertical) {
	for (size_t q = 0; q < c->mapping.length; q++) {
		if (c->mapping.ptr[q].type == CONTROLLER_MAP_AXIS &&
		    c->mapping.ptr[q].axis.side == side &&
		    c->mapping.ptr[q].axis.is_vertical == is_vertical) {
			return c->mapping.ptr[q].axis.uinput_axis;
		}
	}
	return ABS_MAX;
}

static uint8_t calibrated_stick(stick_calibration cal, uint8_t raw) {
	if (raw <= cal.min) {
		return 0x0;
	} else if (raw >= cal.max) {
		return 0xFF;
	} else if (raw <= cal.dead_max && raw >= cal.dead_min) {
		return 0x80;
	} else if (raw <= cal.dead_min) {
		uint8_t raw_range = cal.dead_min - cal.min;
		uint8_t scaled_range = 0x70 - 0x00;
		return ((raw - cal.min) * scaled_range) / raw_range + 0x0;
	} else if (raw >= cal.dead_max) {
		uint8_t raw_range = cal.max - cal.dead_max;
		uint8_t scaled_range = 0xFF - 0x90;
		return ((raw - cal.dead_max) * scaled_range) / raw_range + 0x90;
	} else {
		printf("FATAL: Bad calibration data\n");
		abort();
	}
}

void update_controller(controller_state *c) {
	if ((c->jcl && c->jcl->hidapi_handle == NULL) ||
	    (c->jcr && c->jcr->hidapi_handle == NULL)) {
		joycon_died(c);
		return;
	}

	uint8_t buttons[3];
	memset(buttons, 0, sizeof(buttons));
	if (c->jcl) {
		buttons[0] |= c->jcl->buttons[0];
		buttons[1] |= c->jcl->buttons[1];
		buttons[2] |= c->jcl->buttons[2];
	}
	if (c->jcr) {
		buttons[0] |= c->jcr->buttons[0];
		buttons[1] |= c->jcr->buttons[1];
		buttons[2] |= c->jcr->buttons[2];
	}
	dispatch_buttons(c, buttons, c->prev_button_state);
	memcpy(c->prev_button_state, buttons, 3);
	struct input_event evs[4];
	int evi = 0;
	memset(evs, 0, sizeof(evs));
	if (c->jcl) {
		bool nonzero = false;
		if (c->jcl->stick_v != c->prev_lstick_state[0]) {
			int mapping = get_axis_mapping(c, JC_LEFT, true);
			if (mapping != ABS_MAX) {
				evs[evi].type = EV_ABS;
				evs[evi].code = mapping;
				evs[evi].value =
				    calibrated_stick(c->jcl->calib_v, c->jcl->stick_v);
				if (evs[evi].value != 0x80)
					nonzero = true;
				evi++;
			}
		}
		if (c->jcl->stick_h != c->prev_lstick_state[1]) {
			int mapping = get_axis_mapping(c, JC_LEFT, false);
			if (mapping != ABS_MAX) {
				evs[evi].type = EV_ABS;
				evs[evi].code = mapping;
				evs[evi].value =
				    calibrated_stick(c->jcl->calib_h, c->jcl->stick_h);
				if (evs[evi].value != 0x80)
					nonzero = true;
				evi++;
			}
		}
		if (nonzero)
			printf("Controller #%i: %16s LStick %4d %4d\n", cnum(c), "",
			       -128 + (unsigned int)c->jcl->stick_v,
			       -128 + (unsigned int)c->jcl->stick_h);
		c->prev_lstick_state[0] = c->jcl->stick_v;
		c->prev_lstick_state[1] = c->jcl->stick_h;
	}
	if (c->jcr) {
		bool nonzero = false;
		if (c->jcr->stick_v != c->prev_rstick_state[0]) {
			int mapping = get_axis_mapping(c, JC_RIGHT, true);
			if (mapping != ABS_MAX) {
				evs[evi].type = EV_ABS;
				evs[evi].code = mapping;
				evs[evi].value =
				    calibrated_stick(c->jcr->calib_v, c->jcr->stick_v);
				if (evs[evi].value != 0x80)
					nonzero = true;
				evi++;
			}
		}
		if (c->jcr->stick_h != c->prev_rstick_state[1]) {
			int mapping = get_axis_mapping(c, JC_RIGHT, false);
			if (mapping != ABS_MAX) {
				evs[evi].type = EV_ABS;
				evs[evi].code = mapping;
				evs[evi].value =
				    calibrated_stick(c->jcr->calib_h, c->jcr->stick_h);
				if (evs[evi].value != 0x80)
					nonzero = true;
				evi++;
			}
		}
		if (nonzero)
			printf("Controller #%i: RStick %4d %4d\n", cnum(c),
			       -128 + (unsigned int)c->jcr->stick_v,
			       -128 + (unsigned int)c->jcr->stick_h);
		c->prev_rstick_state[0] = c->jcr->stick_v;
		c->prev_rstick_state[1] = c->jcr->stick_h;
	}
	if (evi > 0) {
		if (write(c->fd, evs, sizeof(evs[0]) * evi) < 0) {
			perror("write uinput");
		}
	}
	memset(evs, 0, sizeof(struct input_event));
	evs[0].type = EV_SYN;
	evs[0].code = SYN_REPORT;
	evs[0].value = 0;
	if (write(c->fd, &evs[0], sizeof(evs[0])) < 0) {
		perror("write uinput");
	}
}

void tick_controller(controller_state *c) {
	if (c->status == CONTROLLER_STATUS_SETUP) {
		setup_controller(c);
		return;
	}
	if (c->status == CONTROLLER_STATUS_TEARDOWN) {
		printf("Removing controller #%i\n", cnum(c));
		destroy_controller(c);
		return;
	}
	if (c->status == CONTROLLER_STATUS_DEADCON) {
		if (((!c->jcl) || (c->jcl && c->jcl->hidapi_handle)) &&
		    ((!c->jcr) || (c->jcr && c->jcr->hidapi_handle))) {
			// Recovered
			c->status = CONTROLLER_STATUS_ACTIVE;
		}
	}
	if (c->status == CONTROLLER_STATUS_ACTIVE) {
		update_controller(c);
	}
}
