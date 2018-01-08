// Copyright Kane York 2018
// Licensed under BSD 2 clause

#ifndef JOYCON_CONTROLLER_HPP
# define JOYCON_CONTROLLER_HPP

# include "Buttons.hpp"
# include "hidapi.h"

namespace joycon {

    class Controller {
    public:
        virtual ~Controller();

        /**
         * @return bitmask of what buttons are pressed
         */
        joycon::Buttons::State GetButtons() const;

        /**
         * Must be called at least 60 times per second, 120 per second for Pro Controller
         */
        virtual void Update() = 0;

    private:
        enum Mode {
            SCANNED, // button push packets
            NORMAL, // 0x30 60hz push
            // GYRO,
            // NFC,
        };

        std::string serial;
        joycon::Side::Enum type;
        Mode mode;

        uint8_t status;
        joycon::Buttons::State buttons;
        uint16_t raw_stick[2][2];
        joycon::Calibration calibration[2];
        // joycon::GyroFrame gyro[3];

        // RGBA8
        uint32_t case_color;
        uint32_t button_color;

        // std::vector<joycon::Rumble::Frame> rumble_queue;
        // int rumble_timer;

        std::vector<std::vector<uint8_t>> subcommand_queue;



        virtual void SendCommand(std::vector<uint8_t> buf) = 0;
    };

    class ControllerBluetooth : Controller {
        virtual ~ControllerBluetooth();

    private:
        hid_device *dev;

        ControllerBluetooth(hid_device *dev, std::string serial, joycon::side::Enum type);

        void SendCommand(std::vector<uint8_t> buf);
    };

    class ControllerUSB : Controller {
        virtual ~ControllerUSB();

    private:
        hid_device *dev;

        ControllerUSB(hid_device *dev, std::string serial, joycon::side::Enum type);

        void SendCommand(std::vector<uint8_t> buf);
    };
}

#endif // JOYCON_CONTROLLER_HPP