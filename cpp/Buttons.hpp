// Copyright Kane York 2018
// Licensed under BSD 2 clause

#ifndef JOYCON_BUTTONS_HPP
# define JOYCON_BUTTONS_HPP

# include <cstdint>
# include <string>
# include <unordered_map>
# include "hidapi.h"

namespace joycon {

    namespace Constants {
        const int VENDOR_NINTENDO = 0x057e;
        const int JOYCON_PRODUCT_L = 0x2006;
        const int JOYCON_PRODUCT_R = 0x2007;
        const int JOYCON_PRODUCT_PRO = 0x2009;
        const int JOYCON_PRODUCT_CHARGEGRIP = 0x200e;
    };

    struct Side {
        enum Enum {
            INVALID = 0,
            LEFT = 1,
            RIGHT = 2,
            BOTH = 3,
        };
    };

    /**
     * @param data 3 bytes of packed 12-bit values
     * @return Unpacked 12-bit values
     */
    inline std::pair<uint16_t, uint16_t> decode_uint12(uint8_t *const data) {
        return std::make_pair(((uint16_t) data[0]) | ((uint16_t) (data[1] & 0xF) << 8),
                              ((uint16_t) data[2] << 4) | ((uint16_t) (data[1] >> 4)));
    };

    /**
     * @param data 3 bytes of packed 12-bit values
     * @return Unpacked 12-bit values
     */
    inline void decode_uint12(uint8_t *const data, uint16_t &d1, uint16_t &d2) {
        d1 = ((uint16_t) data[0]) | ((uint16_t) (data[1] & 0xF) << 8);
        d2 = ((uint16_t) data[2] << 4) | ((uint16_t) (data[1] >> 4));
    };

    struct Buttons {
        typedef uint32_t State;

        enum Enum : State {
            R_Y = 0x01,
            R_X = 0x02,
            R_B = 0x04,
            R_A = 0x08,
            R_SR = 0x10,
            R_SL = 0x20,
            R_R = 0x40,
            R_ZR = 0x80,

            MINUS = 0x0100,
            PLUS = 0x0200,
            R_STICK = 0x0400,
            L_STICK = 0x0800,
            HOME = 0x1000,
            CAPTURE = 0x2000,
            UNUSED_1 = 0x4000,
            UNUSED_2 = 0x8000,

            L_DOWN = 0x010000,
            L_UP = 0x020000,
            L_RIGHT = 0x040000,
            L_LEFT = 0x080000,
            L_SR = 0x100000,
            L_SL = 0x200000,
            L_L = 0x400000,
            L_ZL = 0x800000,
        };

        /*
         * For sideways single joy-con operation, note:
         *
         * Up is R_Y and L_RIGHT
         * Left is R_B and L_UP
         * Right is R_X and L_DOWN
         * Down is R_A and L_LEFT
         *
         * Right trigger is R_SR and L_SR
         * Left trigger is R_SL and L_SL
         *
         * Primary menu/pause is HOME and CAPTURE
         * Secondary menu/pause is PLUS and MINUS
         *
         * Try to avoid requiring the triggers when playing sideways.
         */

        const std::unordered_map<joycon::Buttons::State, std::string> NameMap = {
                {R_Y,      "Y"},
                {R_X,      "X"},
                {R_B,      "B"},
                {R_A,      "A"},
                {R_Y,      "Y"},
                {R_X,      "X"},
                {R_B,      "B"},
                {R_A,      "A"},
                {R_SR,     "R-SR"},
                {R_SL,     "R-SL"},
                {R_R,      "R"},
                {R_ZR,     "ZR"},
                {MINUS,    "-"},
                {PLUS,     "+"},
                {R_STICK,  "RStick"},
                {L_STICK,  "LStick"},
                {HOME,     "Home"},
                {CAPTURE,  "Capture"},
                {UNUSED_1, "Unused1"},
                {UNUSED_2, "Unused2"},
                {L_DOWN,   "Down"},
                {L_UP,     "Up"},
                {L_RIGHT,  "Right"},
                {L_LEFT,   "Left"},
                {L_SR,     "L-SR"},
                {L_SL,     "L-SL"},
                {L_L,      "L"},
                {L_ZL,     "ZL"},
        };

        const joycon::Buttons::State ALL_LEFT = L_DOWN | L_UP | L_RIGHT | L_LEFT |
                                                L_SR | L_SL | L_L | L_ZL |
                                                MINUS | L_STICK | CAPTURE;
        const joycon::Buttons::State ALL_RIGHT = R_Y | R_X | R_B | R_A |
                                                 R_SR | R_SL | R_R | R_ZR |
                                                 PLUS | R_STICK | HOME;
        const joycon::Buttons::State PAIR_SLSR_R = R_SL | R_SR;
        const joycon::Buttons::State PAIR_SLSR_L = L_SL | L_SR;
        const joycon::Buttons::State PAIR_RZR = R_R | R_ZR;
        const joycon::Buttons::State PAIR_LZL = L_L | L_ZL;
        const joycon::Buttons::State PAIR_LR = R_R | L_L;
        const joycon::Buttons::State PAIR_ZLZR = R_ZR | L_ZL;
        const joycon::Buttons::State PAIR_ANY_LR = PAIR_LR | PAIR_ZLZR |
                                                   PAIR_SLSR_R | PAIR_SLSR_L;
    };
}

#endif // JOYCON_BUTTONS_HPP
