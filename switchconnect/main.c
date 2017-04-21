
#define _GNU_SOURCE
#include <arpa/inet.h>
#include <bluetooth/bluetooth.h>
#include <bluetooth/hci.h>
#include <bluetooth/hci_lib.h>
#include <bluetooth/hidp.h>
#include <bluetooth/l2cap.h>
#include <errno.h>
#include <poll.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <unistd.h>

static volatile int done = 0;
static int debug = 1;

/*
 * https://www.bluetooth.org/en-us/specification/assigned-numbers/logical-link-control
 */
#define PSM_SDP 0x0001           // Service Discovery Protocol
#define PSM_HID_Control 0x0011   // Human Interface Device
#define PSM_HID_Interrupt 0x0013 // Human Interface Device

static unsigned short psm_list[] = {
    PSM_SDP, PSM_HID_Control, PSM_HID_Interrupt,
};

#define PSM_MAX_INDEX (sizeof(psm_list) / sizeof(*psm_list))

typedef struct s_proxyconn {
	int which;
	int info_response_cnt;
	// connection request
	int hid_c_scid;
	int hid_i_scid;
	char *adapter;
	char *device;

	struct pollfd *pfd;
	struct s_proxyconn *other;
} t_proxystate;

void terminate(int sig) { done = 1; }

const char *which_from(int which) {
	if (which == 0) {
		return "SWITCH";
	}
	return "JOYCON";
}

const char *which_to(int which) { return which_from(!which); }

static void pfd_close(struct pollfd *p) {
	close(p->fd);
	p->fd = -1;
}

void hexdump(int which, uint8_t *buf, ssize_t len) {
	if (len < 0)
		return;

	for (int i = 0; i < len; i++) {
		printf("%02X ", buf[i]);
		if ((i % 8 == 7) && (i != len)) {
			printf("\n");
		}
	}
	printf("\n");
}

void writedump(int which, int port, char *buf, ssize_t len) {
	if (!debug)
		return;
	if (len < 0)
		return;

	printf("PCKT %d %s > %s: %ld b\n", port, which_from(which), which_to(which), len);
	// TODO - parsing?
	hexdump(which, buf, len);
}

static int is_connected(int fd) {
	int errnum;
	socklen_t sizeof_errnum = sizeof(errnum);
	getsockopt(fd, SOL_SOCKET, SO_ERROR, &errnum, &sizeof_errnum);
	return errnum == 0;
}

static void l2cap_setsockopt(int fd) {
	struct l2cap_options l2o;

	// low security, allow role switch
	int opt = L2CAP_LM_AUTH;
	if (setsockopt(fd, SOL_L2CAP, L2CAP_LM, &opt, sizeof(opt)) < 0) {
		perror("setsockopt L2CAP_LM drop sec");
	}
}

int l2cap_rawconnect(char *adapter, char *device) {
	int fd;
	struct sockaddr_l2 addr;

	fd = socket(AF_BLUETOOTH, SOCK_RAW | SOCK_NONBLOCK, BTPROTO_L2CAP);
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
	str2ba(device, &addr.l2_bdaddr);
	if (connect(fd, (struct sockaddr *)&addr, sizeof(addr)) == -1) {
		if (errno != EINPROGRESS) {
			printf("connect %s: %s\n", device, strerror(errno));
			close(fd);
			return -1;
		}
	}
	return fd;
}

int l2cap_connect(char *adapter, char *device, int psm) {
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
	addr.l2_psm = psm;
	str2ba(device, &addr.l2_bdaddr);
	if (connect(fd, (struct sockaddr *)&addr, sizeof(addr)) == -1) {
		if (errno != EINPROGRESS) {
			printf("connect %s: %s\n", device, strerror(errno));
			close(fd);
			return -1;
		}
	}
	return fd;
}

#define FDI_CNTR 0
#define FDI_HIDC 1
#define FDI_HIDI 2

#define FDI_MAX (1 + FDI_HIDI)

void process_revents(t_proxystate *p);
int wait_for_connection(int c_rawconn, int j_rawconn);

int main(int argc, char **argv) {
	char *c_addr = NULL;
	char *c_adapter = NULL;
	char *j_addr = NULL;
	char *j_adapter = NULL;

	signal(SIGINT, terminate);

	if (argc >= 4) {
		c_addr = argv[1];
		c_adapter = argv[2];
		j_addr = argv[3];
		j_adapter = argv[4];
	}

	if (!c_addr || bachk(c_addr) == -1 ||
	    (c_adapter && (bachk(c_addr) == -1))) {
		printf("usage: %s [console mac addr] [console bt adapter]\n", argv[0]);
		return 1;
	}

	int c_adapter_id = hci_devid(c_adapter);
	int j_adapter_id = hci_devid(j_adapter);
	if (c_adapter_id < 0) {
		printf("failed to get adapter id for %s: %s\n", c_adapter,
		       strerror(errno));
		return 2;
	}
	if (j_adapter_id < 0) {
		printf("failed to get adapter id for %s: %s\n", j_adapter,
		       strerror(errno));
		return 2;
	}
	int c_adapter_dev = hci_open_dev(c_adapter_id);
	if (hci_write_class_of_dev(c_adapter_dev, 0x508, 1000) < 0) {
		printf("failed to write device class: %s\n", strerror(errno));
		return 2;
	}
	int j_adapter_dev = hci_open_dev(j_adapter_id);

	int c_rawconn = l2cap_rawconnect(c_adapter, c_addr);
	int j_rawconn = l2cap_rawconnect(j_adapter, j_addr);

	struct pollfd pfd[2][3];
	t_proxystate cidst[2];
	memset(&pfd, 0, sizeof(pfd));
	memset(&cidst, 0, sizeof(cidst));

	pfd[0][FDI_CNTR].fd = c_rawconn;
	pfd[1][FDI_CNTR].fd = j_rawconn;
	for (int i = 0; i < 2; i++) {
		pfd[i][FDI_CNTR].events = POLLIN;
		pfd[i][FDI_HIDC].fd = -1;
		pfd[i][FDI_HIDI].fd = -1;
		cidst[i].pfd = &pfd[i][0];
		cidst[i].which = i;
	}
	cidst[0].other = &cidst[1];
	cidst[1].other = &cidst[0];
	cidst[0].adapter = c_adapter;
	cidst[0].device = c_addr;
	cidst[1].adapter = j_adapter;
	cidst[1].device = j_addr;

	printf("Connecting...\n");
	usleep(600000);
	if (wait_for_connection(c_rawconn, j_rawconn) < 0) {
		printf("Error: %s\n", strerror(errno));
		return 1;
	}
	printf("Connected\n");

	while (!done) {
		poll(&pfd[0][0], 2 * 3, -1);
		process_revents(&cidst[0]);
		process_revents(&cidst[1]);
	}

	// clean up
}

int wait_for_connection(int c_rawconn, int j_rawconn) {
	struct pollfd pfd[2];
	memset(&pfd, 0, sizeof(pfd));
	pfd[0].fd = c_rawconn;
	pfd[1].fd = j_rawconn;
	pfd[0].events = POLLIN;
	pfd[1].events = POLLIN;
	int allok = 0;

	printf("Connecting...\n");
	while (!done) {
		poll(pfd, 2, -1);
		for (int i = 0; i < 2; i++) {
			if (pfd[i].revents & POLLIN) {
				if (is_connected(pfd[i].fd)) {
					printf("looks like %d is connected\n", i);
					pfd[i].events = 0;
				}
			} else if (pfd[i].revents & (POLLERR | POLLHUP)) {
				int errnum;
				socklen_t sizeof_errnum = sizeof(errnum);
				getsockopt(pfd[i].fd, SOL_SOCKET, SO_ERROR, &errnum,
				           &sizeof_errnum);
				errno = errnum;
				return -1;
			}
		}
		allok = 1;
		for (int i = 0; i < 2; i++) {
			if (pfd[i].events) {
				allok = 0;
			}
		}
		if (allok)
			return 0;
	}
}

void make_hid_connection(t_proxystate *a) {
    return;
    a->pfd[FDI_HIDC].fd = l2cap_connect(a->adapter, a->device, PSM_HID_Control);
    a->pfd[FDI_HIDI].fd = l2cap_connect(a->adapter, a->device, PSM_HID_Interrupt);

    a->pfd[FDI_HIDC].events = POLLIN;
    a->pfd[FDI_HIDI].events = POLLIN;
}

#define READ_MAX 800

void process_revents(t_proxystate *p) {
	char buf[READ_MAX];
	ssize_t len = 0, ret = 0;
	int errnum = 0;

	socklen_t sizeof_errnum = sizeof(errnum);

	for (int fdi = 0; fdi < FDI_MAX; fdi++) {
		if (p->pfd[fdi].revents & (POLLERR | POLLHUP)) {
			getsockopt(p->pfd[fdi].fd, SOL_SOCKET, SO_ERROR, &errnum,
			           &sizeof_errnum);
			printf("poll error: %s\n", strerror(errnum));
			done = 1;
		}
		if (p->pfd[fdi].revents & POLLIN) {
			if (fdi == FDI_CNTR) {
				len = read(p->pfd[fdi].fd, buf, READ_MAX);
				if (len < 0) {
					errnum = errno;
					printf("read %s error: %s\n", which_from(p->which),
					       strerror(errnum));
					done = 1;
					continue;
				}
				writedump(p->which, 0, buf, len);
				ret = 0;
				switch (buf[0]) {
				case L2CAP_CONN_REQ:
					printf("!!! Connection request\n");
					break;
				case L2CAP_CONN_RSP:
					printf("!!! Connection response\n");
					break;
				case L2CAP_INFO_RSP:
					p->info_response_cnt++;
					goto control_packet_default;
				default:
control_packet_default:
					ret = write(p->other->pfd[FDI_CNTR].fd, buf, len);
				}
				if (ret < 0) {
					errnum = errno;
					printf("write %s error: %s\n", which_to(p->which),
					       strerror(errnum));
				}
				if (p->info_response_cnt == 2 && p->other->info_response_cnt == 2) {
				    p->info_response_cnt = -0x300;
				    p->other->info_response_cnt = -0x300;
				    usleep(1000);
				    make_hid_connection(p);
				    make_hid_connection(p->other);
				}
			} else {
				len = read(p->pfd[fdi].fd, buf, READ_MAX);
				if (len < 0) {
					errnum = errno;
					printf("read %s error: %s\n", which_from(p->which),
					       strerror(errnum));
					done = 1;
					continue;
				}
				writedump(p->which, fdi, buf, len);
				ret = write(p->other->pfd[fdi].fd, buf, len);
			}
		}
	}
}
