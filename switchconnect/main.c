
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

static void l2cap_setsockopt(int fd) {
    struct l2cap_options l2o;

    // low security, allow role switch
    int opt = L2CAP_LM_AUTH;
    if (setsockopt(fd, SOL_L2CAP, L2CAP_LM, &opt, sizeof(opt)) < 0) {
	perror("setsockopt L2CAP_LM drop sec");
    }
}

int l2cap_connect(const char *adapter, const char *device, int psm) {
    int fd;
    struct sockaddr_l2 addr;

    fd = socket(AF_BLUETOOTH, SOCK_SEQPACKET | SOCK_NONBLOCK, BTPROTO_L2CAP);
    if (fd == -1) {
	perror("socket");
	return -1;
    }

    l2cap_setsockopt(fd);

    memset(&addr, 0, sizeof(addr));
    addr.l2_family = AF_BLUETOOTH;
    str2ba(adapter, &addr.l2_bdaddr);
    bind(fd, (const struct sockaddr *)&addr, sizeof(addr));

    memset(&addr, 0, sizeof(addr));
    addr.l2_family = AF_BLUETOOTH;
    addr.l2_psm = htobs(psm);
    str2ba(device, &addr.l2_bdaddr);

    if (connect(fd, (struct sockaddr *)&addr, sizeof(addr)) == -1) {
	if (errno != EINPROGRESS) {
	    printf("connect %s %d: %s\n", device, psm, strerror(errno));
	    close(fd);
	    return -1;
	}
    }
    return fd;
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

    struct pollfd pfd[2][PSM_MAX_INDEX];

    int hidc_fd = l2cap_connect(c_adapter, console, PSM_HID_Control);
    pfd[1][1].fd = hidc_fd;
    pfd[1][1].events = POLLOUT;
    poll(&pfd[1][1], 1, -1);
    int hidi_fd = l2cap_connect(c_adapter, console, PSM_HID_Interrupt);
    pfd[1][2].fd = hidi_fd;
    pfd[1][2].events = POLLOUT;
    poll(&pfd[1][2], 1, -1);
    pfd[1][1].events = POLLIN;
    pfd[1][2].events = POLLOUT;


    for(int i = 0; i < 2; i++) {
	for (int psm = 0; psm < PSM_MAX_INDEX; ++psm) {
	    //pfd[i][psm].fd = -1;
	}
    }
    for (int psm = 0; psm < PSM_MAX_INDEX; ++psm) {
	if (psm_list[psm] != PSM_SDP) {
	    //pfd[0][psm].fd = l2cap_connect(c_adapter, console, psm_list[psm]);
	    //pfd[0][psm].events = POLLOUT;
	    //pfd[1][psm].fd = l2cap_connect(j_adapter, joycon, psm_list[psm]);
	    //pfd[1][psm].events = POLLOUT;
	}
    }

#define READ_MAX (256 + 16)
    char buf[READ_MAX];
    ssize_t len, ret;
    int errnum;
    int psm;

    buf[0] = 0x0d;
    buf[1] = 0;
    buf[2] = 0x41; // second dynamic channel number (interrupt)
    buf[3] = 0;
    buf[4] = 0xa1; // HID: DATA, Input
    buf[5] = 0x3f; // button press
    buf[6] = 0x0c;
    buf[7] = 0;
    buf[8] = 9;
    buf[9] = buf[11] = buf[13] = buf[15] = 0;
    buf[10] = buf[12] = buf[14] = buf[16] = 0x80;

    len = write(hidi_fd, buf, 11);
    pfd[1][2].events = POLLOUT;
    poll(&pfd[1][1], 2, -1);
    errnum = 0;
    socklen_t errnum_len = sizeof(errnum);
    getsockopt(pfd[1][2].fd, SOL_SOCKET, SO_ERROR, &errnum, &errnum_len);
    if (errnum != 0)
	perror("write");

    pfd[1][2].events = 0;
    pfd[1][1].events = POLLIN;
    poll(&pfd[1][1], 2, -1);
    len = read(hidc_fd, buf, READ_MAX);
    perror("read");
    writedump(0, buf, len);
    errnum = 0;
    getsockopt(pfd[1][2].fd, SOL_SOCKET, SO_ERROR, &errnum, &errnum_len);
    if (errnum != 0)
	perror("read");

    usleep(50000);
    return 0;

    /*
    int fd = l2cap_connect(c_adapter, console, PSM_SDP);
    pfd[0][0].fd = fd;
    pfd[0][0].events = POLLIN | POLLOUT;
    poll(&pfd[0][0], 1, -1);
    len = read(fd, buf, READ_MAX);
    perror("read");
    writedump(0, buf, len);
    close(fd);
    fd = l2cap_connect(c_adapter, console, PSM_HID_Control);
    pfd[0][0].fd = fd;
    pfd[0][0].events = POLLIN | POLLOUT;
    poll(&pfd[0][0], 1, -1);
    len = read(fd, buf, READ_MAX);
    perror("read");
    writedump(0, buf, len);

    */

    printf("Connection requests sent, starting...\n");
    while (!done) {
	// 0 -> loop again
	if (poll(*pfd, 2 * PSM_MAX_INDEX, -1)) {
	    for (int i = 0; i < 2; i++) {
		for (psm = 0; psm < PSM_MAX_INDEX; ++psm) {
		    if (pfd[i][psm].revents & POLLERR) {
			errnum = 0;
			socklen_t errnum_len = sizeof(errnum);
			getsockopt(pfd[i][psm].fd, SOL_SOCKET, SO_ERROR, &errnum, &errnum_len);
			printf("poll error from %s:%d: %s\n", which_from(i), psm_list[psm], strerror(errnum));
			pfd_close(&pfd[i][psm]);
			//pfd_close(&pfd[1][psm]);
			//done = 1;
			continue;
		    }
		    if (pfd[i][psm].revents & POLLHUP) {
			printf("hangup from %s:%d\n", which_from(i), psm_list[psm]);
			pfd_close(&pfd[0][psm]);
			pfd_close(&pfd[1][psm]);
			done = 1;
			continue;
		    }
		    if (pfd[i][psm].revents & POLLOUT) {
			printf("checking connection %d:%d\n", i, psm);
			errnum = 0;
			socklen_t errnum_len = sizeof(errnum);
			getsockopt(pfd[i][psm].fd, SOL_SOCKET, SO_ERROR, &errnum, &errnum_len);
			if (errnum != EINPROGRESS) {
			    // Connection finished, start listening
			    printf("established connection with %s:%d\n", which_from(i), psm_list[psm]);
			    pfd[i][psm].events = POLLIN;
			}
		    }
		    if (pfd[i][psm].revents & POLLIN) {
			printf("got data on %d:%d\n", i, psm);
			if (pfd[!i][psm].events != POLLIN) {
			    // Ignore, partner not done connecting
			    writedump(i, buf, len);
			    continue;
			}

			len = read(pfd[i][psm].fd, buf, sizeof(buf));
			writedump(i, buf, len);
			if (len > 0) {
			    ret = write(pfd[!i][psm].fd, buf, len);
			    if (ret < 0) {
				errnum = errno;
				printf("write error (%s → %s): %s\n", which_from(i), which_to(i),
					strerror(errnum));
			    }
			} else if (errno != EINTR) {
			    errnum = errno;
			    printf("recv error from %s: %s\n", which_from(i), strerror(errnum));
			    pfd_close(&pfd[0][psm]);
			    pfd_close(&pfd[1][psm]);
			    done = 1;
			    break;
			} else {
			    printf("recv interrupted from %s\n", which_from(i));
			}
		    }
		}
	    }
	}
    }

    for (psm = 0; psm < PSM_MAX_INDEX; psm++) {
	pfd_close(&pfd[0][psm]);
	pfd_close(&pfd[1][psm]);
    }
}
