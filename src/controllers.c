
#include "controllers.h"
#include "joycon.h"
#include <linux/input.h>
#include <stdio.h>
#include <string.h>

joycon_state g_joycons[MAX_JOYCON];
controller_state g_controllers[MAX_OUTCONTROL];
