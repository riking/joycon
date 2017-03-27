
#ifndef CONTROLLERS_H
#define CONTROLLERS_H

#include "joycon.h"

#define CONTROLLER_MAP_EOF 0
#define CONTROLLER_MAP_BUTTON 1
#define CONTROLLER_MAP_AXIS 2
//#define CONTROLLER_MAP_HAT 3

typedef struct cmap_entry {
	int type;
	union {
		struct s_cmap_button {
			jc_button_id joycon_button;
			int uinput_button;
		} button;
		struct s_cmap_axis {
			jc_side side;
			bool is_vertical;
			int uinput_axis;
		} axis;
		struct s_cmap_hat {
			jc_button_id pos;
			jc_button_id neg;
			int uinput_axis;
		} hat;
	};
} cmap_entry;

typedef struct cmap {
	cmap_entry *ptr;
	size_t length;
} cmap;

#define CONTROLLER_STATUS_INACTIVE 0
#define CONTROLLER_STATUS_SETUP 1
#define CONTROLLER_STATUS_ACTIVE 2
#define CONTROLLER_STATUS_PERMERROR 3
#define CONTROLLER_STATUS_DEADCON 4

typedef struct controller {
	int active;
	joycon_state *jcl;
	joycon_state *jcr;
	uint8_t prev_button_state[3];
	uint8_t prev_lstick_state[2];
	uint8_t prev_rstick_state[2];
	cmap mapping;
} controller_state;

#define MAX_JOYCON 10
#define MAX_OUTCONTROL 8

extern joycon_state g_joycons[MAX_JOYCON];
extern controller_state g_controllers[MAX_OUTCONTROL];

extern const cmap cmap_default_two_joycons;
extern const cmap cmap_default_one_joycon;

#endif // CONTROLLERS_H