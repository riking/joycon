
#include "controllers.h"
#include "joycon.h"
#include <linux/input.h>

joycon_state g_joycons[MAX_JOYCON];
controller_state g_controllers[MAX_OUTCONTROL];

static const uint8_t SL_SR = 0xFF & (JC_BUTTON_R_SR | JC_BUTTON_R_SL);

void assign_controller(joycon_state *jc, joycon_state *jc2) {
	if (jc2 == NULL) {
	}
}

void controller_sync_check(void) {
	joycon_state *jc;
	// Check for new controller pairing
	for (int i = 0; i < MAX_JOYCON; i++) {
		jc = &g_joycons[i];
		if (jc->status == JC_ST_WAITING_PAIR) {
			if ((jc->buttons[0] & SL_SR == SL_SR) ||
			    (jc->buttons[2] & SL_SR == SL_SR)) {
				// Pair as single
				printf("Pairing Joy-Con %c %ls as solo controller\n",
				       jc->side == JC_LEFT ? 'L' : 'R', jc->serial);
				jc->status = JC_ST_ACTIVE;
				assign_controller(jc, NULL);
			}
		}
	}
}
