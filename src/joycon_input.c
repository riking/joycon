
#include "joycon.h"
#include "controllers.h"
#include <hidapi/hidapi.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

static void jc_comm_error(joycon_state *jc) {
	if (jc->hidapi_handle) {
		printf("Disconnected from Joy-Con %ls, please reconnect\n", jc->serial);
		jc->hidapi_handle = NULL;
		time(&jc->disconnected_at);
	}
}

void fill_blank_rumble_data(joycon_state *jc, uint8_t *packet) {
	jc->rumble_id++;
	packet[1] = jc->rumble_id & 0xF;
	packet[2] = 0;
	packet[3] = 1;
	packet[4] = 0x40;
	packet[5] = 0x40;
	packet[6] = 0;
	packet[7] = 1;
	packet[8] = 0x40;
	packet[9] = 0x40;
}

void jc_poll_stage1(joycon_state *jc) {
	if (!jc->hidapi_handle) {
		if (jc->disconnected_at + JC_RECONNECT_TIME_MS < time(NULL)) {
			// Reconnect timed out
			printf("Reconnect for Joy-Con %ls timed out, removing\n",
			       jc->serial);
			free(jc->serial);
			memset(jc, 0, sizeof(joycon_state));
			jc->status = JC_ST_INVALID;
			for (int i = 0; i < MAX_OUTCONTROL; i++) {
				if (g_controllers[i].jcl == jc || g_controllers[i].jcr == jc) {
					// Joy-Con expired, so kill the controller
					g_controllers[i].status = CONTROLLER_STATUS_TEARDOWN;
				}
			}
		}
		return;
	}

	// When syncing, we don't need to poll to get stick updates.
	// We can just wait for the button push packets & send the packet then.
	if (jc->status == JC_ST_ACTIVE && jc->outstanding_21_reports < 4) {
		/*
		    uint8_t packet[0x40];
		    memset(packet, 0, sizeof(packet));
		    packet[0] = 1;
		    fill_blank_rumble_data(jc, packet);
		    packet[10] = 0;

		    struct timespec now;
		    clock_gettime(CLOCK_REALTIME, &now);
		    printf("sending %s %d at %lld.%.9ld\n", __FILE__, __LINE__, (long
		   long)now.tv_sec, now.tv_nsec);
		    int res = hid_write((hid_device *)jc->hidapi_handle, packet, 12);
		    if (res < 0) {
		        jc_comm_error(jc);
		        return;
		    }
		    jc->outstanding_21_reports++;
		    */
	}

	if (!jc->did_handshake) {
		jc->did_handshake = true;
		uint8_t packet[0x40];
		memset(packet, 0, sizeof(packet));
		packet[0] = 0x1;
		fill_blank_rumble_data(jc, packet);
		packet[10] = 0x3;
		packet[11] = 0;
		packet[12] = 1;
		printf("sending %s:%d\n", __FILE__, __LINE__);
		int res = hid_write((hid_device *)jc->hidapi_handle, packet, 0x38);
		if (res < 0) {
			jc_comm_error(jc);
		}
		packet[0] = 0x1;
		//        fill_blank_rumble_data(jc, packet);
		packet[10] = 0x38;
		packet[11] = 0;
		printf("sending %s:%d\n", __FILE__, __LINE__);
		res = hid_write((hid_device *)jc->hidapi_handle, packet, 0x38);
		if (res < 0) {
			jc_comm_error(jc);
		}
	}

	// Test - does sending packet 0x10 prevent disconnects?
	int64_t now = time(NULL);
	if ((now - jc->last_packet10_at) > 16) {
		uint8_t packet[0x40];
		memset(packet, 0, sizeof(packet));
		packet[0] = 0x1;
		fill_blank_rumble_data(jc, packet);
		// spi_read 0x6050, len=6
		packet[10] = 38;
		packet[11] = 0xFF;
		//        const uint32_t read_addr = 0x6050;
		//        *((uint32_t*)&packet[11]) = read_addr;
		//         packet[11 + 4] = 6;

		printf("sending %s:%d\n", __FILE__, __LINE__);
		int res = hid_write((hid_device *)jc->hidapi_handle, packet, 0x38);
		if (res < 0) {
			jc_comm_error(jc);
		}
		jc->last_packet10_at = now;
	}
}

static int jc_fill(joycon_state *jc, uint8_t *packet, int len) {
	struct timespec now;
	clock_gettime(CLOCK_REALTIME, &now);
	// printf("got input data at %lld.%.9ld\n", (long long)now.tv_sec,
	// now.tv_nsec);
	(void)len;
	jc->battery = (packet[1] & 0xF0) >> 4;

	jc->buttons[0] = packet[2];
	jc->buttons[1] = packet[3];
	jc->buttons[2] = packet[4];
	if (jc->side == JC_LEFT) {
		// printf("mystery data: %02X %02X\n", packet[5] & 0x0F, packet[6] &
		// 0xF0);
		// packet[5];
		jc->stick_h = ((packet[6] & 0x0F) << 4) | ((packet[5] & 0xF0) >> 4);
		jc->stick_v = packet[7];
	} else {
		// printf("mystery data: %02X %02X\n", packet[8] & 0x0F, packet[9] &
		// 0xF0);
		// packet[8];
		jc->stick_h = ((packet[9] & 0x0F) << 4) | ((packet[8] & 0xF0) >> 4);
		jc->stick_v = packet[10];
	}

	if (packet[12] != 0x80 && packet[12] != 0) {
		// More data
		printf("Got packet:\n");
		int i = 0;
		for (i = 0; i < len; i++) {
			printf("%02X ", packet[i]);
			if (i % 8 == 7)
				printf("\n");
		}
		printf("\n");
		if (packet[13] == 0x10) {
			// SPI Flash read
		}
	}
	return 1;
}

void jc_poll_stage2(joycon_state *jc) {
	uint8_t rbuf[0x60];

	if (!jc->hidapi_handle)
		return;

	bool sent_21 = jc->outstanding_21_reports > 0;
	while (1) {
		int read_res;
		memset(rbuf, 0, 0x60);
		read_res = hid_read_timeout((hid_device *)jc->hidapi_handle, rbuf, 0x60,
		                            JC_READ_TIMEOUT);
		if (read_res < 0) {
			jc_comm_error(jc);
			return;
		} else if (read_res == 0) {
			return;
		}
		if (rbuf[0] != 0x30)
			printf("receive packet id %02X\n", rbuf[0]);
		if (rbuf[0] == 0x21) {
			jc_fill(jc, rbuf + 1, read_res);
		} else if (rbuf[0] == 0x30) {
			jc_fill(jc, rbuf + 1, read_res);
			sent_21 = true;
		} else if (rbuf[0] == 0x3F) {
			// Got button update, request an update if we aren't waiting for one
			if (!sent_21) {
				uint8_t packet[0x40];
				memset(packet, 0, sizeof(packet));
				packet[0] = 0x1;
				packet[10] = 1;

				printf("sending %s:%d\n", __FILE__, __LINE__);
				int res =
				    hid_write((hid_device *)jc->hidapi_handle, packet, 0x40);
				if (res < 0) {
					jc_comm_error(jc);
					return;
				}
				//				jc->outstanding_21_reports++;
				sent_21 = true;
			}
			continue;
		} else {
			dprintf(2, "WARNING: Joycon %ls sent HID packet %02X\n", jc->serial,
			        rbuf[0]);
			continue;
		}
	}
	return;
}

bool jc_getbutton(jc_button_id bid, joycon_state *jc) {
	int byte_num = ((bid & 0x0F00) >> 8);
	if (byte_num < 2 || byte_num > 4) {
		return false;
	}
	return (jc->buttons[byte_num - 2] & (bid & 0xFF)) != 0;
}

bool jc_getbutton2(jc_button_id bid, joycon_state *jcl, joycon_state *jcr) {
	int byte_num = ((bid & 0x0F00) >> 8);
	if (byte_num == 2 || (bid == JC_BUTTON_R_STI || bid == JC_BUTTON_R_HOM ||
	                      bid == JC_BUTTON_R_PLU)) {
		return jc_getbutton(bid, jcr);
	}
	if (byte_num == 4 || (bid == JC_BUTTON_L_STI || bid == JC_BUTTON_L_CAP ||
	                      bid == JC_BUTTON_L_MIN)) {
		return jc_getbutton(bid, jcl);
	}
	return 0;
}

struct jc_button_name_map {
	jc_button_id bid;
	const char *name;
};

static const struct jc_button_name_map jc_button_name_map[] = {
    {JC_BUTTON_R_Y, "Y"},        {JC_BUTTON_R_X, "X"},
    {JC_BUTTON_R_B, "B"},        {JC_BUTTON_R_A, "A"},
    {JC_BUTTON_R_SR, "R-SR"},    {JC_BUTTON_R_SL, "R-SL"},
    {JC_BUTTON_R_R, "R"},        {JC_BUTTON_R_ZR, "ZR"},
    {JC_BUTTON_L_MIN, "-"},      {JC_BUTTON_R_PLU, "+"},
    {JC_BUTTON_R_STI, "RStick"}, {JC_BUTTON_L_STI, "LStick"},
    {JC_BUTTON_R_HOM, "Home"},   {JC_BUTTON_L_CAP, "Capture"},
    {JC_BUTTON_L_DD, "Down"},    {JC_BUTTON_L_DU, "Up"},
    {JC_BUTTON_L_DR, "Right"},   {JC_BUTTON_L_DL, "Left"},
    {JC_BUTTON_L_SR, "L-SR"},    {JC_BUTTON_L_SL, "L-SL"},
    {JC_BUTTON_L_L, "L"},        {JC_BUTTON_L_ZL, "ZL"},
};

const char *jc_button_name(jc_button_id bid) {
	for (size_t i = 0;
	     i < (sizeof(jc_button_name_map) / sizeof(jc_button_name_map[0]));
	     i++) {
		if (jc_button_name_map[i].bid == bid) {
			return jc_button_name_map[i].name;
		}
	}
	return "";
}

jc_button_id jc_button_byname(char *str) {
	for (size_t i = 0;
	     i < (sizeof(jc_button_name_map) / sizeof(jc_button_name_map[0]));
	     i++) {
		if (0 == strcmp(str, jc_button_name_map[i].name)) {
			return jc_button_name_map[i].bid;
		}
	}
	return 0;
}
