// Copyright Kane York 2018
// Licensed under BSD 2 clause

#ifndef JOYCON_CALIBRATION_H
#define JOYCON_CALIBRATION_H

#include <cstdint>
#include <vector>
#include "Buttons.hpp"

namespace joycon {

    class Calibration {
    public:
        Calibration();
        /**
         * @param side Which stick this calibration is for; must be LEFT or RIGHT.
         * @param data Byte data.
         */
        Calibration(joycon::Side::Enum side, std::vector<uint8_t> data);
        Calibration(Calibration& const src);
        virtual ~Calibration();
        Calibration &operator=(Calibration const &rhs);

        /**
         * Convert raw stick data into calibrated stick data.
         * @param raw_stick Raw 12-bit stick values from the input.
         * @return Values are in the range (-0x7FF, +0x7FF).
         */
        std::pair<int16_t, int16_t> Adjust(std::pair<uint16_t, uint16_t> raw_stick) const;

    private:
        uint16_t xCenter;
        uint16_t yCenter;
        uint16_t xMinOff;
        uint16_t yMinOff;
        uint16_t xMaxOff;
        uint16_t yMaxOff;
    };
}

#endif // JOYCON_CALIBRATION_H
