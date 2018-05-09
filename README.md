# joycon
Joy-Con input driver for Linux

The current main code, written in Go, lives in the `prog4/jcdriver` folder, as it was the fourth attempt at making this program.

## Compiling

```
sudo apt install libudev-dev
go get -u github.com/riking/joycon/prog4/jcdriver || true
```

## Basic Instructions

After starting the program as root, connect the Joy-Cons over Bluetooth. Once the program has picked them
up, the player lights will begin flashing. Press L+R on the joycons to "pair" them up as a single controller device,
which is then available to the rest of the system.

If you want the joycons to function as a pair of controllers with analog sticks, press the SL + SR buttons to pair as a single controller.

TODO: Interface to switch between the modes / drop controllers for re-pairing

## Limitations

The code will not actively scan and connect to the joycons. When the code is run on Mac, the "press button to
reconnect" works due to the OS bluetooth driver (though the code cannot provide the device to other programs,
as that requires a kext).

TODO: Scan and reconnect mode for bluez.

## TODO List

PRs and help for any of this is appreciated!
If I neglect your contributions, try pinging me on Twitter (@riking27); I may have missed the notification.

 - Output codes for "Held down HOME" and "Held down CAPTURE", and possibly "Pressed HOME" / "Pressed CAPTURE"
   - Ref: [Subcommand 4](https://github.com/dekuNukem/Nintendo_Switch_Reverse_Engineering/blob/master/bluetooth_hid_subcommands_notes.md#subcommand-0x04-trigger-buttons-elapsed-time)
 - Graphical controller management interface
   - Programmatic interface, so someone else can build the graphical interface
   - custom Capture button handling, possibly?
 - Linux: accept reconnect requests from the controllers
 - "Active scanning" mode to pick up new controllers without holding down SYNC button
 - Configuration storage - save calibration data so it doesn't need to be pulled on every connection
 - Button and **stick axis** remapping - some games want stick inverted
 - Motion control support
    - Figure out how to work the insane system Linux has of delivering gyro data. Consider requiring use of a custom protocol.
 - Implement as a C++ library
    - Implement as a C library
    - Linux kernel driver to remove the suppressed devices so games don't try to use them
    - Mac kernel driver to present the combined controller to other programs
    - Windows kernel driver to do anything at all with the controllers
 
