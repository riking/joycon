
#include "controllers.h"
#include "joycon.h"

void scan_joycons(void);

int main(int argc, char *argv[]) {
	int res;
	scan_joycons();
}