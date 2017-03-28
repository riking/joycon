
#include "controllers.h"
#include "joycon.h"
#include "loop.h"

#include <stdio.h>
#include <time.h>

static int tick;

static int ticks_with_unpaired;

static void mainloop(void) {

	// Attach new devices
	if (tick == 0) {
		scan_joycons();
	}
	tick++;
	if (tick == 60)
		tick = 0;

	// Poll for input
	for (int i = 0; i < MAX_JOYCON; i++) {
		if (g_joycons[i].status != 0) {
			jc_poll_stage1(&g_joycons[i]);
		}
	}
	// Receive input
	for (int i = 0; i < MAX_JOYCON; i++) {
		if (g_joycons[i].status != 0) {
			jc_poll_stage2(&g_joycons[i]);
		}
	}
	// Pair new controllers
	bool any_unpaired = false;
	for (int i = 0; i < MAX_JOYCON; i++) {
		if (g_joycons[i].status == JC_ST_WAITING_PAIR) {
			attempt_pairing(&g_joycons[i]);
		}
		if (g_joycons[i].status == JC_ST_WAITING_PAIR) {
			any_unpaired = true;
		}
	}
	if (any_unpaired) {
		ticks_with_unpaired++;
	} else {
		ticks_with_unpaired = 0;
	}
	if (ticks_with_unpaired > 60 * 3) {
		printf("\nPress L and R on the controller to set up the Joy-Con\n");
		ticks_with_unpaired = 60 * -60;
	}
	// Update controllers
	for (int i = 0; i < MAX_OUTCONTROL; i++) {
		if (g_controllers[i].status != 0) {
			tick_controller(&g_controllers[i]);
		}
	}
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

void setup_controller(controller_state *c);
void destroy_controller(controller_state *c) ;

int main(void) {
	struct timespec sleep_target;
	struct timespec cycle_end;
	struct timespec remaining;

/*
	g_controllers[0].mapping = cmap_default_one_joycon;
	setup_controller(&g_controllers[0]);

	sleep(10);

	destroy_controller(&g_controllers[0]);
	*/

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
		// Run program
		// pthread_mutex_lock();
		mainloop();
		// pthread_mutex_unlock();
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