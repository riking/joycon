
SRCFILES = attach_devices.c controllers.c joycon_input.c main.c
SRCFILES += uinput_keys.c mapping.c uinput_keys.c calibration.c crc.c
HEADFILES = controllers.h joycon.h uinput_keys.h

CFLAGS = -Wall -Wextra -Wmissing-prototypes
CFLAGS += $(shell pkg-config --cflags hidapi-hidraw)

LDFLAGS = -Wall -Wextra
LDFLAGS += $(shell pkg-config --libs hidapi-hidraw)

ifdef DEBUG
	CFLAGS += -fsanitize=address -g
	LDFLAGS += -fsanitize=address -g
endif

SRCS = $(addprefix src/, $(SRCFILES))
HEADS = $(addprefix src/, $(HEADFILES))
OBJS = $(SRCS:.c=.o)

all: jcmapper

format: $(SRCS) $(HEADS) switchconnect/main.c
	clang-format -style=file -i $^

clean:
	rm -f $(OBJS)
	rm -f devinput/hidapi_demo.o
	rm -f switchconnect/main.o

jcmapper: $(OBJS)
	gcc -o $@ $^ $(LDFLAGS)

jcreader: devinput/hidapi_demo.c
	gcc -o $@ devinput/hidapi_demo.c $(shell pkg-config --libs hidapi-hidraw) -fsanitize=address -g

swprobe: switchconnect/main.c
	gcc -o $@ $^ -lbluetooth

%.o: %.c $(HEADS)
	gcc -c -o $@ $< $(CFLAGS)
