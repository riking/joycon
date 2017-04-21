
#define _GNU_SOURCE
#include <stdlib.h>
#include <errno.h>
#include <sys/socket.h>
#include <bluetooth/bluetooth.h>
#include <bluetooth/l2cap.h>
#include <bluetooth/hidp.h>
#include <bluetooth/hci.h>
#include <bluetooth/hci_lib.h>
#include <stdio.h>
#include <sys/ioctl.h>
#include <poll.h>
#include <unistd.h>
#include <arpa/inet.h>

static volatile int done = 0;
static int debug = 1;

/*
 * https://www.bluetooth.org/en-us/specification/assigned-numbers/logical-link-control
 */
#define PSM_SDP 0x0001 //Service Discovery Protocol
#define PSM_RFCOMM  0x0003 //Can't be used for L2CAP sockets
#define PSM_TCS_BIN 0x0005 //Telephony Control Specification
#define PSM_TCS_BIN_CORDLESS  0x0007 //Telephony Control Specification
#define PSM_BNEP  0x000F //Bluetooth Network Encapsulation Protocol
#define PSM_HID_Control 0x0011 //Human Interface Device
#define PSM_HID_Interrupt 0x0013 //Human Interface Device
#define PSM_UPnP  0x0015
#define PSM_AVCTP 0x0017 //Audio/Video Control Transport Protocol
#define PSM_AVDTP 0x0019 //Audio/Video Distribution Transport Protocol
#define PSM_AVCTP_Browsing  0x001B //Audio/Video Remote Control Profile
#define PSM_UDI_C_Plane 0x001D //Unrestricted Digital Information Profile
#define PSM_ATT 0x001F
#define PSM_3DSP 0x0021 //3D Synchronization Profile

static unsigned short psm_list[] =
{
    PSM_SDP,
    //PSM_TCS_BIN,
    //PSM_TCS_BIN_CORDLESS,
    //PSM_BNEP,
    PSM_HID_Control,
    PSM_HID_Interrupt,
    0xff, // HID control
    //PSM_UPnP,
    //PSM_AVCTP,
    //PSM_AVDTP,
    //PSM_AVCTP_Browsing,
    //PSM_UDI_C_Plane,
    //PSM_ATT,
    //PSM_3DSP
};

#define PSM_MAX_INDEX (sizeof(psm_list)/sizeof(*psm_list))

void terminate(int sig)
{
    done = 1;
}

const char *which_from(int which) {
    if (which == 0) {
	return "SWITCH";
    }
    return "JOYCON";
}

const char *which_to(int which) {
    return which_from(!which);
}

static void pfd_close(struct pollfd *p) {
    close(p->fd);
    p->fd = -1;
}

void hexdump(int which, uint8_t *buf, ssize_t len) {
    if (len < 0) return;

    for (int i = 0; i < len; i++) {
	printf("%02X ", buf[i]);
	if ((i % 8 == 7) && (i != len)) {
	    printf("\n");
	}
    }
    printf("\n");
}

void writedump(int which, char *buf, ssize_t len) {
    if (!debug) return;
    if (len < 0) return;

    printf("PCKT %s > %s: %ld\n", which_from(which), which_to(which), len);
    // TODO - parsing?
    hexdump(which, buf, len);
}

int main(int argc, char **argv) {
    char *console = NULL;
    char *c_adapter = NULL;
    char *joycon = NULL;
    char *j_adapter = NULL;

    if (argc >= 4)
    {
	console = argv[1];
	c_adapter = argv[2];
	joycon = argv[3];
	j_adapter = argv[4];
    }


    if (!console || bachk(console) == -1 || (c_adapter && (bachk(console) == -1))) {
	printf("usage: %s [console mac addr] [console bt adapter]\n", argv[0]);
	return 1;
    }

    int c_adapter_id = hci_devid(c_adapter);
    int j_adapter_id = hci_devid(j_adapter);
    if (c_adapter_id < 0 || j_adapter_id < 0) {
	printf("failed to get adapter id: %s\n", strerror(errno));
	return 2;
    }
    int c_adapter_dev = hci_open_dev(c_adapter_id);
    if (hci_write_class_of_dev(c_adapter_dev, 0x508, 1000) < 0) {
	printf("failed to write device class: %s\n", strerror(errno));
	return 2;
    }
    int j_adapter_dev = hci_open_dev(j_adapter_id);


}
