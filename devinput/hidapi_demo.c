#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <stdint.h>
#include <string.h>
#include <errno.h>
#include <wchar.h>
#include <unistd.h>

#include <hidapi/hidapi.h>

#define JOYCON_VENDOR 0x057e
#define JOYCON_PRODUCT_L 0x2006
#define JOYCON_PRODUCT_R 0x2007

#define SERIAL_LEN 18

typedef struct s_joycon {
    wchar_t     *serial;
    hid_device  *handle;
    int         left_right; // 1: left, 2: right
    int         controller_id; // -1: unassigned
    uint16_t    buttons;
    int8_t      dstick; // TODO get analog stick
}               t_joycon;

t_joycon        g_joycons[8];

// joycon_1 is R, joycon_2 is L
#define CONTROLLER_TYPE_BOTH  0x1
// joycon_1 is L, joycon_2 is R
#define CONTROLLER_TYPE_LONLY 0x2
// joycon_1 is R, joycon_2 is -1
#define CONTROLLER_TYPE_RONLY 0x3

#define L_OR_R(lr) (lr == 1 ? 'L' : (lr == 2 ? 'R' : '?'))

typedef struct  s_controller {
    int     type;
    int     joycon_1_num;
    int     joycon_2_num;
}               t_controller;

t_controller    g_controllers[4];

void    found_joycon(struct hid_device_info *dev) {
    t_joycon *jc = NULL;
    int i = 0;

    for (; i < 8; i++) {
        if (g_joycons[i].left_right == 0) {
            jc = &g_joycons[i];
            break;
        }
    }
    if (jc) {
        if (dev->product_id == JOYCON_PRODUCT_L) {
            jc->left_right = 1;
        } else {
            jc->left_right = 2;
        }
        jc->serial = wcsdup(dev->serial_number);
        jc->buttons = 0;
        jc->controller_id = -1;
        printf("Found joycon %c %i: %ls %s\n",
            L_OR_R(jc->left_right), i, jc->serial, dev->path);
        errno = 0;
        jc->handle = hid_open_path(dev->path);
        hid_set_nonblocking(jc->handle, 1);
        if (jc->handle == NULL) {
            printf("Could not open serial %ls: %s\n", g_joycons[i].serial, strerror(errno));
        }
    }
}

struct s_button_map {
    int bit;
    char *name;
};

struct s_button_map button_map[16] = {
    {0, "D"},
    {1, "R"},
    {2, "L"},
    {3, "U"},
    {4, "SL"},
    {5, "SR"},
    {6, "?"},
    {7, "?"},
    {8, "-"},
    {9, "+"},
    {10, "LS"},
    {11, "RS"},
    {12, "Ho"},
    {13, "Sc"},
    {14, "LR"},
    {15, "ZLR"},
};

void    print_buttons(t_joycon *jc)
{
    if (jc->controller_id == -1) {
        printf("Joycon %c (Unattached): ", L_OR_R(jc->left_right));
    } else {
        printf("Joycon %c (Controller %d): ", L_OR_R(jc->left_right), jc->controller_id);
    }

    for (int i = 0; i < 16; i++) {
        if (jc->buttons & (1 << button_map[i].bit)) {
            printf("\033[0m\033[1m");
        } else {
            printf("\033[0m");
        }
        printf("%s", button_map[i].name);
        printf("\033[0m ");
    }
    printf("\n");
}

const char * const dstick_names[9] = {"Up", "UR", "Ri", "DR", "Do", "DL", "Le", "UL", "Neu"};

void print_dstick(t_joycon *jc) {
    if (jc->controller_id == -1) {
        printf("Joycon %c (Unattached): ", L_OR_R(jc->left_right));
    } else {
        printf("Joycon %c (Controller %d): ", L_OR_R(jc->left_right), jc->controller_id);
    }

    printf("%s\n", dstick_names[jc->dstick]);
}

void    handle_input(t_joycon *jc, uint8_t *buf65, int len)
{
    switch (buf65[0]) {
    case 63:
        ;
        uint16_t old_buttons = jc->buttons;
        int8_t old_dstick = jc->dstick;
        // button update
        jc->buttons = buf65[1] + buf65[2] * 256;
        jc->dstick = buf65[3];
        if (jc->buttons != old_buttons) {
            print_buttons(jc);
        }
        if (jc->dstick != old_dstick) {
            print_dstick(jc);
        }
        break;
    case 0x21:
        // Button status:
        // Byte 1: 0x8E
        //  Byte 2
        //   Bit 0: JR Y
        //   Bit 1: JR X
        //   Bit 2: JR B
        //   Bit 3: JR A
        //   Bit 4: JR SR
        //   Bit 5: JR SL
        //   Bit 6: JR R
        //   Bit 7: JR ZR
        // Byte 4
        //   Bit 0: JL Down
        //   Bit 1: JL Up
        //   Bit 2: JL Right
        //   Bit 3: JL Left
        //   Bit 4: JL SR
        //   Bit 5: JL SL
        //   Bit 6: JL L
        //   Bit 7: JL ZL
        // Byte 3
        //   Bit 2: RStick
        //   Bit 3: LStick
        //   Bit 4: Home
        //   Bit 5: Capture
        if (jc->controller_id == -1) {
            printf("Joycon %c (Unattached): ", L_OR_R(jc->left_right));
        } else {
            printf("Joycon %c (Controller %d): ", L_OR_R(jc->left_right), jc->controller_id);
        }
        printf("Packet 0x21 Status\n");
        if (len != 6 * 8 + 1) {
            printf("[!!!] Different length for packet 0x21\n");
        }
        for (int i = 1; i < len; i++) {
            printf("%02X ", (uint8_t)buf65[i]);
            if (i % 8 == 0) printf("\n");
        }
        uint8_t *pckt = buf65 + 2;
        if (pckt[0] != 0x8E) {
            printf("[!!!] Button status not starting with 0x8E\n");
        } else {
            printf("Button status: ");
            for (int i = 1; i < 4; i++) {
                for (int b = 0; b < 8; b++) {
                    char c = (pckt[i] & (1 << b)) ? '1' : '0';
                    printf("%c", c);
                }
                printf(" ");
            }
            printf("\n");
        }
        uint8_t *stick_data;
        if (jc->left_right == 1) {
            stick_data = pckt + 4;
            //printf("Stick L: %02X %02X %02X\n", pckt[4], pckt[5], pckt[6]);
        } else {
            stick_data = pckt + 7;
            //printf("Stick R: %02X %02X %02X\n", pckt[7], pckt[8], pckt[9]);
        }
        uint8_t stick_unk = stick_data[0];
        uint8_t stick_hz = ((stick_data[1] & 0x0F) << 4) | ((stick_data[1] & 0xF0) >> 4);
        uint8_t stick_vert = stick_data[2];
        printf("Stick %c: [%02X] %d %d\n", L_OR_R(jc->left_right), stick_unk, -128 + (int)(unsigned int)stick_hz, -128 + (int)(unsigned int)stick_vert);
        printf("Unknown: %02X %02X %02X %02X\n", pckt[10], pckt[11], pckt[12], pckt[13]);
        break;
    default:
        printf("Unknown packet: ");
        for (int i = 0; i < len; i++) {
            printf("%02X ", buf65[i]);
        }
        printf("\n");
    }
}

int main(int argc, char* argv[])
{
	int res;
	unsigned char buf[65];
	#define MAX_STR 255
	wchar_t wstr[MAX_STR];
	hid_device *handle;
	int i;

	// Enumerate and print the HID devices on the system
	struct hid_device_info *devs, *cur_dev;

	devs = hid_enumerate(0x0, 0x0);
	cur_dev = devs;

	while (cur_dev) {
	    if (cur_dev->vendor_id == JOYCON_VENDOR) {
	        if (cur_dev->product_id == JOYCON_PRODUCT_L || cur_dev->product_id == JOYCON_PRODUCT_R) {
	            found_joycon(cur_dev);
	        }
	    }
		cur_dev = cur_dev->next;
	}
	hid_free_enumeration(devs);

    t_joycon *jc;
    for (;;) {
        for (int jci = 0; jci < 8; jci++) {
            jc = &g_joycons[jci];
            if (!jc->handle) continue;


            res = hid_read(jc->handle, buf, 65);
            if (res < 0)
            {
                printf("Unable to read from joycon %i, disconnecting\n", jci);
                jc->handle = 0;
                jc->left_right = 0;
            }
            else if (res > 0)
            {
            // writing ID 1 causes it to reply with a packet 0x21 (33)

                uint16_t old_buttons = jc->buttons;
                handle_input(jc, buf, res);
                if (!(old_buttons & (1 << 14)) && (jc->buttons & (1 << 14))) {
                    memset(buf, 0, 65);
                    buf[0] = 0x01;
                    buf[1] = 0x91;
                    buf[2] = 0x11;
                    buf[3] = 0;
                    buf[4] = 0;
                    buf[5] = 0;
                    buf[6] = 0;
                    buf[7] = 0;
                    buf[8] = 0;
                    errno = 0;
                    res = hid_write(jc->handle, buf, 9);
                    if (res < 0)
                    {
                        printf("write error: %s\n", strerror(errno));
                    }
                }
                if (!(old_buttons & (1 << 15)) && (jc->buttons & (1 << 15))) {
                    memset(buf, 0, 65);
                    buf[0] = 0x01;
                    buf[1] = 0;
                    buf[2] = 0x92;
                    buf[3] = 0x00;
                    buf[4] = 0x00;
                    buf[5] = 0x01;
                    buf[6] = 0;
                    buf[7] = 0;
                    buf[8] = 0x69;
                    buf[9] = 0x2d;
                    buf[10] = 0;
                    buf[11] = 0;
                    buf[12] = 0;
                    errno = 0;
                    res = hid_write(jc->handle, buf, 9);
                    if (res < 0)
                    {
                        printf("write error: %s\n", strerror(errno));
                    }
                }
            }

        }
        usleep(15);
    }

return 0;

    if (!handle)
    {
        perror("");
    }
    else
    {
        // Read the Manufacturer String
        res = hid_get_manufacturer_string(handle, wstr, MAX_STR);
        printf("Manufacturer String: %ls\n", wstr);

        // Read the Product String
        res = hid_get_product_string(handle, wstr, MAX_STR);
        printf("Product String: %ls\n", wstr);

        // Read the Serial Number String
        res = hid_get_serial_number_string(handle, wstr, MAX_STR);
        printf("Serial Number String: %ls", wstr);
        printf("\n");

        memset(buf, 0, 65);
        buf[0] = 0x2;
        errno = 0;
        res = hid_send_feature_report(handle, buf, 9);
        if (errno) {
            printf("Feature Report %i: %s\n", buf[0], strerror(errno));
            return 1;
        }

        char nbuf[257];
        memset(nbuf, 0, 257);
        nbuf[0] = 0x2;
        res = hid_get_feature_report(handle, nbuf, 9);
        if (errno) {
            printf("get Feature Report %i: %s\n", nbuf[0], strerror(errno));
            return 1;
        }

        // Print out the returned buffer.
        printf("Feature Report %i\n   ", buf[0]);
        for (i = 0; i < res; i++)
            printf("%02hhx ", buf[i]);
        printf("\n");

        return 0;

        // Set the hid_read() function to be non-blocking.

        // Send an Output report to toggle the LED (cmd 0x80)
        buf[0] = 1; // First byte is report number
        buf[1] = 0x80;
        res = hid_write(handle, buf, 65);

        // Send an Output report to request the state (cmd 0x81)
        buf[1] = 0x81;
        hid_write(handle, buf, 65);

        // Read requested state
        res = hid_read(handle, buf, 65);
        if (res < 0)
            printf("Unable to read()\n");

        // Print out the returned buffer.
        for (i = 0; i < res; i++)
            printf("buf[%d]: %d\n", i, buf[i]);

    }
	return 0;
}
