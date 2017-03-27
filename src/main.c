
#include "controllers.h"
#include "joycon.h"
#include "loop.h"

#include <stdio.h>
#include <time.h>

static int tick;

static void mainloop(void) {
	// Attach new devices
	if (tick % 60 == 0) {
		scan_joycons();
	}
	tick++;

	// Poll for input
	for (int i = 0; i < MAX_JOYCON; i++) {
		if (g_joycons[i].status == JC_ST_WAITING_PAIR ||
		    g_joycons[i].status == JC_ST_CALIBRATING ||
		    g_joycons[i].status == JC_ST_ACTIVE) {
			jc_poll_stage1(&g_joycons[i]);
		}
	}
	for (int i = 0; i < MAX_JOYCON; i++) {
		if (g_joycons[i].status == JC_ST_WAITING_PAIR ||
		    g_joycons[i].status == JC_ST_CALIBRATING ||
		    g_joycons[i].status == JC_ST_ACTIVE) {
			jc_poll_stage2(&g_joycons[i]);
		}
	}

	controller_pairing_check();
}

#define BILLION 1000000000
/* https://www.gnu.org/software/libc/manual/html_node/Elapsed-Time.html */
static int timespec_subtract(struct timespec *result, struct timespec *x,
                             struct timespec *y) {
	/* Perform the carry for the later subtraction by updating y. */
	if (x->tv_nsec < y->tv_nsec) {
		int secs = (y->tv_nsec - x->tv_nsec) / BILLION + 1;
		y->tv_nsec -= BILLION * secs;
		y->tv_sec += secs;
	}
	if (x->tv_nsec - y->tv_nsec > BILLION) {
		int secs = (x->tv_nsec - y->tv_nsec) / BILLION;
		y->tv_nsec += BILLION * secs;
		y->tv_sec -= secs;
	}

	/* Compute the time remaining to wait.
	   tv_usec is certainly positive. */
	result->tv_sec = x->tv_sec - y->tv_sec;
	result->tv_nsec = x->tv_nsec - y->tv_nsec;

	/* Return 1 if result is negative. */
	if (x->tv_sec == y->tv_sec) {
		return x->tv_nsec < y->tv_nsec;
	}
	return x->tv_sec < y->tv_sec;
}

int main(int argc, char *argv[]) {
	struct timespec sleep_target;
	struct timespec cycle_end;
	struct timespec remaining;

	while (1) {
		// Compute now + 1/60 second
		clock_gettime(CLOCK_MONOTONIC, &sleep_target);
		uint64_t nsec = sleep_target.tv_nsec;
		nsec += 16.60 * (BILLION / 1000LL);
		if (nsec > BILLION) {
			sleep_target.tv_nsec = nsec - BILLION;
			sleep_target.tv_sec += 1;
		} else {
			sleep_target.tv_nsec = nsec;
		}
		printf("Will poll %ld\n", sleep_target.tv_nsec);
		// Run program
		mainloop();
		// Sleep until 15ms elapsed
		while (1) {
			clock_gettime(CLOCK_MONOTONIC, &cycle_end);
			if (timespec_subtract(&remaining, &sleep_target, &cycle_end)) {
				break; // sleep_target < cycle_end
			}
			nanosleep(&remaining, NULL);
		}
	}
}