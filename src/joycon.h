
#ifndef JOYCON_H
#define JOYCON_H

#include <stdbool.h>
#include <stdint.h>
#include <wchar.h>

#define JOYCON_VENDOR 0x057e
#define JOYCON_PRODUCT_L 0x2006
#define JOYCON_PRODUCT_R 0x2007

#define SERIAL_LEN 18

#ifndef JC_READ_TIMEOUT
#define JC_READ_TIMEOUT 2
#endif

#define JC_RECONNECT_TIME_MS 30 * 1000

typedef enum e_jc_side { JC_SIDE_INVALID, JC_LEFT, JC_RIGHT } jc_side;

// state transitions:
//   Invalid -> Waiting_Pair on device detected
//   Waiting_Pair -> Calibrating on start calibration
//   Calibrating -> Waiting_Pair on finish calibration
//   Waiting_Pair -> Active on controller assign
//   Active -> Waiting_Pair on controller teardown
//   Any -> Invalid on reconnect timeout
typedef enum jc_status {
	JC_ST_INVALID,
	JC_ST_WAITING_PAIR,
	JC_ST_CALIBRATING,
	JC_ST_ACTIVE
} jc_status;

typedef struct {
	uint8_t _is_default;
	uint8_t neutral;
	uint8_t dead_down;
	uint8_t dead_up;
	uint8_t min;
	uint8_t max;
} stick_calibration;

typedef struct {
	wchar_t *serial;
	stick_calibration vertical;
	stick_calibration horizontal;
} calibration_data;

typedef struct s_joycon_state {
	wchar_t *serial;
	void *hidapi_handle;
	jc_side side;
	jc_status status;
	int64_t disconnected_at;

	uint8_t stick_v;
	uint8_t stick_h;
	uint8_t buttons[3];

	int outstanding_21_reports;
	stick_calibration calib_v;
	stick_calibration calib_h;
} joycon_state;

// byte number (2-4), bit number (mask & 0xFF)
#define JC_BUTTON_R_Y 0x0201
#define JC_BUTTON_R_X 0x0202
#define JC_BUTTON_R_B 0x0204
#define JC_BUTTON_R_A 0x0208
#define JC_BUTTON_R_SR 0x0210
#define JC_BUTTON_R_SL 0x0220
#define JC_BUTTON_R_R 0x0240
#define JC_BUTTON_R_ZR 0x0280

#define JC_BUTTON_L_MIN 0x0301
#define JC_BUTTON_R_PLU 0x0302
#define JC_BUTTON_R_STI 0x0304
#define JC_BUTTON_L_STI 0x0308
#define JC_BUTTON_R_HOM 0x0310
#define JC_BUTTON_L_CAP 0x0320

#define JC_BUTTON_L_DD 0x0401
#define JC_BUTTON_L_DU 0x0402
#define JC_BUTTON_L_DR 0x0404
#define JC_BUTTON_L_DL 0x0408
#define JC_BUTTON_L_SR 0x0410
#define JC_BUTTON_L_SL 0x0420
#define JC_BUTTON_L_L 0x0440
#define JC_BUTTON_L_ZL 0x0480

typedef uint16_t jc_button_id;

void jc_poll_stage1(joycon_state *jc);
void jc_poll_stage2(joycon_state *jc);

bool jc_getbutton(jc_button_id bid, joycon_state *jc);
bool jc_getbutton2(jc_button_id bid, joycon_state *jcl, joycon_state *jcr);

const char *jc_button_name(jc_button_id bid);
jc_button_id jc_button_byname(char *str);

calibration_data calibration_file_load(wchar_t *serial);
int calibration_file_save(wchar_t *serial, calibration_data data);

#endif // JOYCON_H