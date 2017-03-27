
#include "controllers.h"
#include "joycon.h"
#include <linux/input.h>

joycon_state g_joycons[MAX_JOYCON];
controller_state g_controllers[MAX_OUTCONTROL];

void controller_sync_check(void) {
	// Check for new controller pairing
	for (int i = 0; i < MAX_JOYCON; i++) {
		if (g_joycons[i].status == JC_ST_WAITING_PAIR) {

		}
	}
}
