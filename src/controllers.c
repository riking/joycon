
#include "controllers.h"
#include "joycon.h"

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

	struct uinput_user_dev uidev;
	memset(&uidev, 0, sizeof(uidev));

	if (c->jcl && c->jcr) {
		snprintf(uidev.name, UINPUT_MAX_NAME_SIZE, "Joy-Con Pair #%i", cnum(c));
	} else if (c->jcl) {
		snprintf(uidev.name, UINPUT_MAX_NAME_SIZE, "Joy-Con L #%i", cnum(c));
	} else {
		snprintf(uidev.name, UINPUT_MAX_NAME_SIZE, "Joy-Con R #%i", cnum(c));
	}
	uidev.id.bustype = BUS_VIRTUAL;
	uidev.id.vendor = JOYCON_VENDOR;
	uidev.id.product = 0x1FF8; // uhhhh just make something up
	uidev.id.version = 1;

	// note: because of this, need to re-init controller whenever config changes
	for (size_t i = 0; i < c->mapping.length; i++) {
		if (c->mapping.ptr[i].type == CONTROLLER_MAP_AXIS) {
			int axis = c->mapping.ptr[i].axis.uinput_axis;
			ret = ioctl(c->fd, UI_SET_ABSBIT, axis);
			if (ret < 0) {
				printf("Error starting controller %i: uinput error: %s\n",
				       cnum(c), strerror(errno));
				c->status = CONTROLLER_STATUS_TEARDOWN;
				return;
			}
			uidev.absmin[axis] = 0;
			uidev.absmax[axis] = 0xFF;
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
	    write(c->fd, ev, sizeof(ev[0]) * evi);
	}
}

static void joycon_died(controller_state *c) {
	c->status = CONTROLLER_STATUS_DEADCON;
	memset(c->prev_button_state, 0, sizeof(c->prev_button_state));
	memset(c->prev_lstick_state, 0, sizeof(c->prev_lstick_state));
	memset(c->prev_rstick_state, 0, sizeof(c->prev_rstick_state));
}

void update_controller(controller_state *c) {
	if ((c->jcl && c->jcl->hidapi_handle == NULL) ||
	    (c->jcr && c->jcr->hidapi_handle == NULL)) {
		joycon_died(c);
		return;
	}

	uint8_t buttons[3];
	uint8_t lstick[2];
	uint8_t rstick[2];
	memset(buttons, 0, sizeof(buttons));
	memset(lstick, 0, sizeof(lstick));
	memset(rstick, 0, sizeof(rstick));
	if (c->jcl) {
		buttons[0] |= c->jcl->buttons[0];
		buttons[1] |= c->jcl->buttons[1];
		buttons[2] |= c->jcl->buttons[2];
		lstick[0] = c->jcl->stick_v;
		lstick[1] = c->jcl->stick_h;
	}
	if (c->jcr) {
		buttons[0] |= c->jcr->buttons[0];
		buttons[1] |= c->jcr->buttons[1];
		buttons[2] |= c->jcr->buttons[2];
		rstick[0] = c->jcr->stick_v;
		rstick[1] = c->jcr->stick_h;
	}
	dispatch_buttons(c, buttons, c->prev_button_state);
	c->prev_button_state[0] = buttons[0];
	c->prev_button_state[1] = buttons[1];
	c->prev_button_state[2] = buttons[2];
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