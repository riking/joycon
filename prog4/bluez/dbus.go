package bluez

import "github.com/godbus/dbus"
import "github.com/godbus/dbus/introspect"

//Properties dbus serializable struct
// Use struct tags to control how the field is handled by Properties interface
// Example: field `dbus:writable,emit,myCallback`
// See Prop in github.com/godbus/dbus/prop for configuration details
// Options:
// - writable: set the property as writable (Set will updated it). Omit for read-only
// - emit|invalidates: emit PropertyChanged, invalidates emit without disclosing the value. Omit for read-only
// - callback: a callable function in the struct compatible with the signature of Prop.Callback. Omit for no callback
type Properties interface {
	ToMap() (map[string]interface{}, error)
}

const (

	//Device1Interface the bluez interface for Device1
	Device1Interface = "org.bluez.Device1"
	//Adapter1Interface the bluez interface for Adapter1
	Adapter1Interface = "org.bluez.Adapter1"
	//GattService1Interface the bluez interface for GattService1
	GattService1Interface = "org.bluez.GattService1"
	//GattCharacteristic1Interface the bluez interface for GattCharacteristic1
	GattCharacteristic1Interface = "org.bluez.GattCharacteristic1"
	//GattDescriptor1Interface the bluez interface for GattDescriptor1
	GattDescriptor1Interface = "org.bluez.GattDescriptor1"

	//ObjectManagerInterface the dbus object manager interface
	ObjectManagerInterface = "org.freedesktop.DBus.ObjectManager"
	//InterfacesRemoved the DBus signal member for InterfacesRemoved
	InterfacesRemoved = "org.freedesktop.DBus.ObjectManager.InterfacesRemoved"
	//InterfacesAdded the DBus signal member for InterfacesAdded
	InterfacesAdded = "org.freedesktop.DBus.ObjectManager.InterfacesAdded"

	//PropertiesInterface the DBus properties interface
	PropertiesInterface = "org.freedesktop.DBus.Properties"
	//PropertiesChanged the DBus properties interface and member
	PropertiesChanged = "org.freedesktop.DBus.Properties.PropertiesChanged"
)

// ObjectManagerIntrospectDataString introspect ObjectManager description
const ObjectManagerIntrospectDataString = `
<interface name="org.freedesktop.DBus.ObjectManager">
	<method name="GetManagedObjects">
		<arg name="objects" type="a{oa{sa{sv}}}" direction="out" />
	</method>
	<signal name="InterfacesAdded">
		<arg name="object" type="o"/>
		<arg name="interfaces" type="a{sa{sv}}"/>
	</signal>
	<signal name="InterfacesRemoved">
		<arg name="object" type="o"/>
		<arg name="interfaces" type="as"/>
	</signal>
</interface>`

// ObjectManagerIntrospectData introspect ObjectManager description
var ObjectManagerIntrospectData = introspect.Interface{
	Name: "org.freedesktop.DBus.ObjectManager",
	Methods: []introspect.Method{
		{
			Name: "GetManagedObjects",
			Args: []introspect.Arg{
				{
					Name:      "objects",
					Type:      "a{oa{sa{sv}}}",
					Direction: "out",
				},
			},
		},
	},
	Signals: []introspect.Signal{
		{
			Name: "InterfacesAdded",
			Args: []introspect.Arg{
				{
					Name: "object",
					Type: "o",
				},
				{
					Name: "interfaces",
					Type: "a{sa{sv}}",
				},
			},
		},
		{
			Name: "InterfacesRemoved",
			Args: []introspect.Arg{
				{
					Name: "object",
					Type: "o",
				},
				{
					Name: "interfaces",
					Type: "as",
				},
			},
		},
	},
}

// GattService1IntrospectDataString interface definition
const GattService1IntrospectDataString = `
<interface name="org.bluez.GattService1">
  <property name="UUID" type="s" access="read"></property>
  <property name="Device" type="o" access="read"></property>
  <property name="Primary" type="b" access="read"></property>
  <property name="Characteristics" type="ao" access="read"></property>
</interface>
`

// GattDescriptor1IntrospectDataString interface definition
const GattDescriptor1IntrospectDataString = `
<interface name="org.bluez.GattDescriptor1">
  <method name="ReadValue">
    <arg name="value" type="ay" direction="out"/>
  </method>
  <method name="WriteValue">
    <arg name="value" type="ay" direction="in"/>
  </method>
  <property name="UUID" type="s" access="read"></property>
  <property name="Characteristic" type="o" access="read"></property>
  <property name="Value" type="ay" access="read"></property>
</interface>
`

//GattCharacteristic1IntrospectDataString interface definition
const GattCharacteristic1IntrospectDataString = `
<interface name="org.bluez.GattCharacteristic1">
  <method name="ReadValue">
    <arg name="value" type="ay" direction="out"/>
  </method>
  <method name="WriteValue">
    <arg name="value" type="ay" direction="in"/>
  </method>
  <method name="StartNotify"></method>
  <method name="StopNotify"></method>
  <property name="UUID" type="s" access="read"></property>
  <property name="Service" type="o" access="read"></property>
  <property name="Value" type="ay" access="read"></property>
  <property name="Notifying" type="b" access="read"></property>
  <property name="Flags" type="as" access="read"></property>
  <property name="Descriptors" type="ao" access="read"></property>
</interface>
`

//Device1IntrospectDataString interface definition
const Device1IntrospectDataString = `
<interface name="org.bluez.Device1">
  <method name="Disconnect"></method>
  <method name="Connect"></method>
  <method name="ConnectProfile">
    <arg name="UUID" type="s" direction="in"/>
  </method>
  <method name="DisconnectProfile">
    <arg name="UUID" type="s" direction="in"/>
  </method>
  <method name="Pair"></method>
  <method name="CancelPairing"></method>
  <property name="Address" type="s" access="read"></property>
  <property name="Name" type="s" access="read"></property>
  <property name="Alias" type="s" access="readwrite"></property>
  <property name="Class" type="u" access="read"></property>
  <property name="Appearance" type="q" access="read"></property>
  <property name="Icon" type="s" access="read"></property>
  <property name="Paired" type="b" access="read"></property>
  <property name="Trusted" type="b" access="readwrite"></property>
  <property name="Blocked" type="b" access="readwrite"></property>
  <property name="LegacyPairing" type="b" access="read"></property>
  <property name="RSSI" type="n" access="read"></property>
  <property name="Connected" type="b" access="read"></property>
  <property name="UUIDs" type="as" access="read"></property>
  <property name="Modalias" type="s" access="read"></property>
  <property name="Adapter" type="o" access="read"></property>
  <property name="ManufacturerData" type="a{qv}" access="read"></property>
  <property name="ServiceData" type="a{sv}" access="read"></property>
  <property name="TxPower" type="n" access="read"></property>
  <property name="GattServices" type="ao" access="read"></property>
</interface>
`

// GattService1IntrospectData interface definition
var GattService1IntrospectData = introspect.Interface{
	Name: "org.bluez.GattService1",
	Properties: []introspect.Property{
		{
			Name:   "UUID",
			Access: "read",
			Type:   "s",
		},
		{
			Name:   "Device",
			Access: "read",
			Type:   "o",
		},
		{
			Name:   "Primary",
			Access: "read",
			Type:   "b",
		},
		{
			Name:   "Characteristics",
			Access: "read",
			Type:   "ao",
		},
	},
}

// Defines how the characteristic value can be used. See
// Core spec "Table 3.5: Characteristic Properties bit
// field", and "Table 3.8: Characteristic Extended
// Properties bit field"
const (
	FlagCharacteristicBroadcast                 = "broadcast"
	FlagCharacteristicRead                      = "read"
	FlagCharacteristicWriteWithoutResponse      = "write-without-response"
	FlagCharacteristicWrite                     = "write"
	FlagCharacteristicNotify                    = "notify"
	FlagCharacteristicIndicate                  = "indicate"
	FlagCharacteristicAuthenticatedSignedWrites = "authenticated-signed-writes"
	FlagCharacteristicReliableWrite             = "reliable-write"
	FlagCharacteristicWritableAuxiliaries       = "writable-auxiliaries"
	FlagCharacteristicEncryptRead               = "encrypt-read"
	FlagCharacteristicEncryptWrite              = "encrypt-write"
	FlagCharacteristicEncryptAuthenticatedRead  = "encrypt-authenticated-read"
	FlagCharacteristicEncryptAuthenticatedWrite = "encrypt-authenticated-write"
	FlagCharacteristicSecureRead                = "secure-read"
	FlagCharacteristicSecureWrite               = "secure-write"
)

// Descriptor specific flags
const (
	FlagDescriptorRead                      = "read"
	FlagDescriptorWrite                     = "write"
	FlagDescriptorEncryptRead               = "encrypt-read"
	FlagDescriptorEncryptWrite              = "encrypt-write"
	FlagDescriptorEncryptAuthenticatedRead  = "encrypt-authenticated-read"
	FlagDescriptorEncryptAuthenticatedWrite = "encrypt-authenticated-write"
	FlagDescriptorSecureRead                = "secure-read"
	FlagDescriptorSecureWrite               = "secure-write"
)
