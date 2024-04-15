package config_exporter

import (
	"fmt"
	"github.com/go-kid/ioc/app"
	"github.com/go-kid/ioc/component_definition"
	"github.com/go-kid/ioc/configure"
	"github.com/go-kid/ioc/definition"
	"github.com/go-kid/ioc/factory"
	"github.com/go-kid/ioc/factory/processors"
	"github.com/go-kid/ioc/syslog"
	"github.com/go-kid/ioc/util/mode"
	"github.com/go-kid/ioc/util/properties"
	"github.com/go-kid/ioc/util/reflectx"
	"gopkg.in/yaml.v3"
	"reflect"
)

type postProcessor struct {
	processors.DefaultInstantiationAwareComponentPostProcessor
	definition.PriorityComponent
	configure          configure.Configure
	properties         []*component_definition.Property
	propertyOriginArgs map[string]component_definition.TagArg
}

func (d *postProcessor) PostProcessComponentFactory(factory factory.Factory) error {
	d.configure = factory.GetConfigure()
	return nil
}

func NewConfigExporter() ConfigExporter {
	return &postProcessor{
		propertyOriginArgs: make(map[string]component_definition.TagArg),
	}
}

func (d *postProcessor) Order() int {
	return -1
}

func copyArg(arg component_definition.TagArg) component_definition.TagArg {
	copied := component_definition.TagArg{}
	for argType, strings := range arg {
		copied[argType] = strings
	}
	return copied
}

func (d *postProcessor) PostProcessBeforeInstantiation(m *component_definition.Meta, componentName string) (any, error) {
	if _, ok := m.Raw.(*app.App); ok {
		return m.Raw, nil
	}
	for _, prop := range m.GetAllProperties() {
		if prop.PropertyType == component_definition.PropertyTypeConfiguration {
			d.propertyOriginArgs[prop.Field.ID()+prop.Tag] = copyArg(prop.Args())
			d.properties = append(d.properties, prop)
		}
		prop.SetArg(component_definition.ArgRequired, "false")
	}
	return nil, nil
}

func (d *postProcessor) PostProcessBeforeInitialization(component any, componentName string) (any, error) {
	return nil, nil
}

func (d *postProcessor) ForEachConfiguration(f func(property *component_definition.Property, prefix string, val any)) {
	for _, property := range d.properties {
		for p, a := range property.Configurations {
			prefix := p
			value := a
			tagArg := d.propertyOriginArgs[property.Field.ID()+property.Tag]
			for argType, strings := range tagArg {
				property.SetArg(argType, strings...)
			}
			f(property, prefix, value)
		}
	}
}

func (d *postProcessor) GetConfig(mode mode.Mode) properties.Properties {
	pm := properties.New()
	d.ForEachConfiguration(func(property *component_definition.Property, prefix string, value any) {
		if value == nil {
			value = reflectx.ZeroValue(property.Type)
		}

		if mode.Eq(AnnotationArgs) {
			property.Args().ForEach(func(argType component_definition.ArgType, args []string) {
				if len(args) == 0 || (len(args) == 1 && args[0] == "") {
					pm.Set(fmt.Sprintf("%s@Args.%s", prefix, argType), true)
				} else {
					pm.Set(fmt.Sprintf("%s@Args.%s", prefix, argType), args)
				}
			})
		}

		if mode.Eq(AnnotationSource | AnnotationSourceProperty) {
			var source string
			if mode.Eq(AnnotationSourceProperty) {
				source = property.String()
			} else {
				source = property.Holder.String()
			}
			annoPath := fmt.Sprintf("%s@Sources", prefix)
			if sources, ok := pm.Get(annoPath); ok {
				pm.Set(annoPath, append(sources.([]string), source))
			} else {
				pm.Set(annoPath, []string{source})
			}
		}

		if origin := d.configure.Get(prefix); origin != nil {
			if mode.Eq(OnlyNew) {
				return
			}
			if mode.Eq(Append) {
				value = origin
			}
		}

		setProperties(property, pm, prefix, value)
	})
	return pm
}

func setProperties(property *component_definition.Property, pm properties.Properties, prefix string, value any) {
	switch property.Type.Kind() {
	case reflect.Struct, reflect.Map:
		deepSet(prefix, value, pm)
	case reflect.Pointer:
		if eleKind := property.Type.Elem().Kind(); eleKind == reflect.Struct || eleKind == reflect.Map {
			deepSet(prefix, value, pm)
			return
		}
		fallthrough
	default:
		pm.Set(prefix, value)
	}
}

func deepSet(prefix string, value any, pm properties.Properties) {
	subRaw := toMap(value)
	subProp := properties.NewFromMap(subRaw)
	for subP, subAnyVal := range subProp {
		pm.Set(prefix+"."+subP, subAnyVal)
	}
}

func toMap(a any) map[string]any {
	bytes, err := yaml.Marshal(a)
	if err != nil {
		syslog.Panicf("yaml marshal error: %#v, %+v", a, err)
	}
	var subRaw = make(map[string]any)
	err = yaml.Unmarshal(bytes, subRaw)
	if err != nil {
		syslog.Panicf("yaml unmarshal error: %s, %+v", string(bytes), err)
	}
	return subRaw
}
