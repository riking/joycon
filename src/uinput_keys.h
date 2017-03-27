
#ifndef UINPUT_KEYS_H
#define UINPUT_KEYS_H

// returns NULL on failure
const char *uinput_key_name(int key);

// returns KEY_MAX on failure
int uinput_key_byname(char *name);

// returns NULL on failure
const char *uinput_axis_name(int axis);

// returns ABS_MAX on failure
int uinput_axis_byname(char *name);

#endif // UINPUT_KEYS_H
