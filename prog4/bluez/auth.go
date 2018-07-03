package bluez

import (
	"fmt"
	"reflect"

	"github.com/godbus/dbus"
	"github.com/godbus/dbus/introspect"
	"github.com/pkg/errors"
)

var agentIntrospectData = introspect.Node{
	Interfaces: []introspect.Interface{
		introspect.IntrospectData,
		agent1IntrospectData,
	},
	Children: nil,
}

var rootIntrospectData = introspect.Node{
	Interfaces: []introspect.Interface{
		introspect.IntrospectData,
	},
	Children: []introspect.Node{
		{Name: "bluezagent"},
	},
}

var dbusSignatureUint32 = dbus.SignatureOfType(reflect.TypeOf(uint32(0))).String()
var dbusSignatureUint16 = dbus.SignatureOfType(reflect.TypeOf(uint16(0))).String()

var agent1IntrospectData = introspect.Interface{
	Name: "org.bluez.Agent1",
	Methods: []introspect.Method{
		{
			Name: "Release",
		},
		{
			Name: "RequestPinCode",
			Args: []introspect.Arg{
				{"device", "o", "in"},
				{"pincode", "s", "out"},
			},
		},
		{
			Name: "DisplayPinCode",
			Args: []introspect.Arg{
				{"device", "o", "in"},
				{"pincode", "s", "in"},
			},
		},
		{
			Name: "RequestPasskey",
			Args: []introspect.Arg{
				{"device", "o", "in"},
				{"passkey", dbusSignatureUint32, "out"},
			},
		},
		{
			Name: "DisplayPasskey",
			Args: []introspect.Arg{
				{"device", "o", "in"},
				{"passkey", dbusSignatureUint32, "in"},
				{"entered", dbusSignatureUint16, "in"},
			},
		},
		{
			Name: "RequestConfirmation",
			Args: []introspect.Arg{
				{"device", "o", "in"},
				{"passkey", dbusSignatureUint32, "in"},
			},
		},
		{
			Name: "RequestAuthorization",
			Args: []introspect.Arg{
				{"device", "o", "in"},
			},
		},
		{
			Name: "AuthorizeService",
			Args: []introspect.Arg{
				{"device", "o", "in"},
				{"uuid", "s", "in"},
			},
		},
		{
			Name: "Cancel",
		},
	},
}

type DBusAgent struct {
	introspect.Introspectable
}

func (a *JoyconAPI) registerAgent() error {
	agent := &DBusAgent{
		Introspectable: introspect.NewIntrospectable(&agentIntrospectData),
	}
	var err error
	err = a.busConn.Export(agent, dbus.ObjectPath("/bluezagent"), "org.bluez.Agent1")
	if err != nil {
		return err
	}
	err = a.busConn.Export(agent, dbus.ObjectPath("/bluezagent"), "org.freedesktop.DBus.Introspectable")
	if err != nil {
		return err
	}
	err = a.busConn.Export(&DBusRoot{
		Introspectable: introspect.NewIntrospectable(&rootIntrospectData),
	}, dbus.ObjectPath("/"), "org.freedesktop.DBus.Introspectable")
	if err != nil {
		return err
	}

	obj := a.busConn.Object(BlueZBusName, "/org/bluez")
	call := obj.Call("org.bluez.AgentManager1.RegisterAgent", 0,
		dbus.ObjectPath("/bluezagent"),
		"KeyboardDisplay",
	)
	if call.Err != nil {
		return errors.Wrap(call.Err, "register bluetooth pairing agent")
	}
	return nil
}

func (b *DBusAgent) Release() *dbus.Error {
	fmt.Println("agent Release")
	return nil
}

func (b *DBusAgent) RequestPinCode(device dbus.ObjectPath) (string, *dbus.Error) {
	fmt.Println("agent RequestPinCode", device)
	return "0000", nil
}

func (b *DBusAgent) DisplayPinCode(device dbus.ObjectPath, pincode string) *dbus.Error {
	fmt.Println("agent DisplayPinCode", device, pincode)
	return nil
}

func (b *DBusAgent) RequestPasskey(device dbus.ObjectPath, pincode string) (uint32, *dbus.Error) {
	fmt.Println("agent RequestPasskey", device, pincode)
	return 0, nil
}

func (b *DBusAgent) DisplayPasskey(device dbus.ObjectPath, passkey uint32, entered uint16) *dbus.Error {
	fmt.Println("agent DisplayPasskey", device, passkey, "entered", entered)
	return nil
}

func (b *DBusAgent) RequestConfirmation(device dbus.ObjectPath, passkey uint32) *dbus.Error {
	fmt.Println("agent RequestConfirmation", device, passkey)
	return nil
}

func (b *DBusAgent) RequestAuthorization(device dbus.ObjectPath) *dbus.Error {
	fmt.Println("agent RequestAuthorization", device)
	return nil
}

func (b *DBusAgent) AuthorizeService(device dbus.ObjectPath, uuid string) *dbus.Error {
	fmt.Println("agent AuthorizeService", device, uuid)
	if uuid == HIDProfileUUIDStr || uuid == ProfileX1UUIDStr || uuid == ProfileX2UUIDStr {
		return nil
	}
	return &dbus.Error{Name: "org.bluez.Error.Rejected"}
	return nil
}

func (b *DBusAgent) Cancel() *dbus.Error {
	fmt.Println("agent Cancel")
	return nil
}

type DBusRoot struct {
	introspect.Introspectable
}
