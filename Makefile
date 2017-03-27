
SRCFILES = attach_devices.c controllers.c joycon_input.c main.c
SRCFILES += uinput_keys.c
HEADFILES = controllers.h joycon.h

CFLAGS = -Wall -Wextra
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

format: $(SRCS) $(HEADS)
	clang-format -style=file -i $^

jcmapper: $(OBJS)
	gcc -o $@ $^ $(LDFLAGS)

jcreader: devinput/hidapi_demo.c
	gcc -o $@ devinput/hidapi_demo.c $(shell pkg-config --libs hidapi-hidraw) -fsanitize=address -g

%.o: %.c
	gcc -c -o $@ $^ $(CFLAGS)
