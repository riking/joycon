
#include "controllers.h"
#include <linux/input.h>

static cmap_entry default_two_joycons[] = {
    /* abxy, dpad */
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_A, .uinput_button = BTN_EAST,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_X, .uinput_button = BTN_NORTH,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_B, .uinput_button = BTN_SOUTH,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_Y, .uinput_button = BTN_WEST,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_DL, .uinput_button = BTN_DPAD_LEFT,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_DD, .uinput_button = BTN_DPAD_DOWN,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_DU, .uinput_button = BTN_DPAD_UP,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_DR, .uinput_button = BTN_DPAD_RIGHT,
         }},
    /* shoulder */
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_L, .uinput_button = BTN_TL,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_ZL, .uinput_button = BTN_TL2,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_R, .uinput_button = BTN_TR,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_ZR, .uinput_button = BTN_TR2,
         }},
    /* plus & minus */
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_MIN, .uinput_button = BTN_SELECT,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_PLU, .uinput_button = BTN_START,
         }},
    /* home & capture */
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_HOM, .uinput_button = BTN_MODE,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_CAP, .uinput_button = KEY_SYSRQ,
         }},
    /* sticks */
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_STI, .uinput_button = BTN_THUMBL,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_STI, .uinput_button = BTN_THUMBR,
         }},
    {.type = CONTROLLER_MAP_AXIS,
     .axis =
         {
             .side = JC_LEFT, .is_vertical = false, .uinput_axis = ABS_Z,
         }},
    {.type = CONTROLLER_MAP_AXIS,
     .axis =
         {
             .side = JC_LEFT, .is_vertical = true, .uinput_axis = ABS_RX,
         }},
    {.type = CONTROLLER_MAP_AXIS,
     .axis =
         {
             .side = JC_RIGHT, .is_vertical = false, .uinput_axis = ABS_RY,
         }},
    {.type = CONTROLLER_MAP_AXIS,
     .axis =
         {
             .side = JC_RIGHT, .is_vertical = true, .uinput_axis = ABS_RZ,
         }},
    {.type = CONTROLLER_MAP_EOF}};

// note: the buttons are rotated for the 1-joycon grip
static cmap_entry default_one_joycon[] = {
    /* abxy left */
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_DL, .uinput_button = BTN_SOUTH,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_DD, .uinput_button = BTN_EAST,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_DU, .uinput_button = BTN_WEST,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_DR, .uinput_button = BTN_NORTH,
         }},
    /* abxy right */
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_A, .uinput_button = BTN_SOUTH,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_X, .uinput_button = BTN_EAST,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_B, .uinput_button = BTN_WEST,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_Y, .uinput_button = BTN_NORTH,
         }},
    /* sl/sr */
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_SL, .uinput_button = BTN_TL,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_SL, .uinput_button = BTN_TL,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_SR, .uinput_button = BTN_TR,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_SR, .uinput_button = BTN_TR,
         }},
    /* side shoulder */
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_L, .uinput_button = BTN_C,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_R, .uinput_button = BTN_C,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_ZL, .uinput_button = BTN_Z,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_ZR, .uinput_button = BTN_Z,
         }},
    /* plus & minus */
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_MIN, .uinput_button = BTN_START,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_PLU, .uinput_button = BTN_START,
         }},
    /* home & capture */
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_HOM, .uinput_button = BTN_MODE,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_CAP, .uinput_button = BTN_MODE,
         }},
    /* sticks */
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_L_STI, .uinput_button = BTN_THUMBL,
         }},
    {.type = CONTROLLER_MAP_BUTTON,
     .button =
         {
             .joycon_button = JC_BUTTON_R_STI, .uinput_button = BTN_THUMBL,
         }},
    /* note: controller.c will ignore entries for the joycon this isn't */
    {.type = CONTROLLER_MAP_AXIS,
     .axis =
         {
             .side = JC_LEFT, .is_vertical = true, .uinput_axis = ABS_Z,
         }},
    {.type = CONTROLLER_MAP_AXIS,
     .axis =
         {
             .side = JC_LEFT, .is_vertical = false, .uinput_axis = ABS_RX,
         }},
    {.type = CONTROLLER_MAP_AXIS,
     .axis =
         {
             .side = JC_RIGHT, .is_vertical = true, .uinput_axis = ABS_Z,
         }},
    {.type = CONTROLLER_MAP_AXIS,
     .axis =
         {
             .side = JC_RIGHT, .is_vertical = false, .uinput_axis = ABS_RX,
         }},
    {.type = CONTROLLER_MAP_EOF}};

const cmap cmap_default_two_joycons = {
    .ptr = &default_two_joycons[0],
    .length = sizeof(default_two_joycons) / sizeof(default_two_joycons[0]),
};

const cmap cmap_default_one_joycon = {
    .ptr = &default_one_joycon[0],
    .length = sizeof(default_one_joycon) / sizeof(default_one_joycon[0]),
};
