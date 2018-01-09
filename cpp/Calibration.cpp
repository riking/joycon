// Copyright Kane York 2018
// Licensed under BSD 2 clause

#include "Calibration.hpp"
#include "Buttons.hpp"


joycon::Calibration::Calibration() : xMaxOff(0x400), xMinOff(0x400), xCenter(0x800), yCenter(0x800), yMaxOff(0x400), yMinOff(0x400) {
}

joycon::Calibration::Calibration(joycon::Side::Enum side, std::vector<uint8_t> data) {
    if (side == joycon::Side::LEFT) {
        joycon::decode_uint12(data.data() + 3, this->xCenter, this->yCenter);
        joycon::decode_uint12(data.data() + 6, this->xMinOff, this->yMinOff);
        joycon::decode_uint12(data.data() + 0, this->xMaxOff, this->yMaxOff);
    } else {
        joycon::decode_uint12(data.data() + 0, this->xCenter, this->yCenter);
        joycon::decode_uint12(data.data() + 3, this->xMinOff, this->yMinOff);
        joycon::decode_uint12(data.data() + 6, this->xMaxOff, this->yMaxOff);
    }
}

joycon::Calibration::Calibration(joycon::Calibration& const src) : xMaxOff(src.xMaxOff), xMinOff(src.xMinOff),
                                                                   xCenter(src.xCenter), yCenter(src.yCenter),
                                                                   yMaxOff(src.yMaxOff), yMinOff(src.yMinOff) {
}

joycon::Calibration::~Calibration() {}

joycon::Calibration &joycon::Calibration::operator=(const joycon::Calibration &rhs) {
    this->xCenter = rhs.xCenter;
    this->yCenter = rhs.yCenter;
    this->xMinOff = rhs.xMinOff;
    this->yMinOff = rhs.yMinOff;
    this->xMaxOff = rhs.xMaxOff;
    this->yMaxOff = rhs.yMaxOff;
    return *this;
}

std::pair<int16_t, int16_t> joycon::Calibration::Adjust(std::pair<uint16_t, uint16_t> raw_stick) const {
    int16_t xOut;
    int16_t yOut;

    xOut = (int16_t) raw_stick.first - (int16_t) this->xCenter;
    yOut = (int16_t) raw_stick.second - (int16_t) this->yCenter;

    if (xOut < 0) {
        xOut = (int16_t)((((double)xOut) * 0x7FF) / (double)this->xMinOff);
    } else {
        xOut = (int16_t)((((double)xOut) * 0x7FF) / (double)this->xMaxOff);
    }
    if (yOut < 0) {
        yOut = (int16_t)((((double)yOut) * 0x7FF) / (double)this->yMinOff);
    } else {
        yOut = (int16_t)((((double)yOut) * 0x7FF) / (double)this->yMaxOff);
    }

    int32_t magnitude = xOut * (int32_t) xOut + yOut * (int32_t) yOut;
    if (magnitude > (0x7FF * 0x7FF)) {
        double reduceFactor = sqrt(((double)(0x7FF * 0x7FF)) / magnitude);
        xOut = (int16_t) (xOut * reduceFactor);
        yOut = (int16_t) (yOut * reduceFactor);
    }

    return std::pair<int16_t, int16_t>(xOut, yOut);
}
