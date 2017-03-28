
#ifndef LOOP_H
#define LOOP_H

void scan_joycons(void);
void jc_poll_stage1(joycon_state *jc);
void jc_poll_stage2(joycon_state *jc);
void attempt_pairing(joycon_state *jc);
void tick_controller(controller_state *c);

#endif // LOOP_H
