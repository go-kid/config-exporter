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
	"github.com/go-kid/ioc/util/el"
	"github.com/go-kid/ioc/util/mode"
	"github.com/go-kid/ioc/util/properties"
	"github.com/go-kid/ioc/util/reflectx"
	"gopkg.in/yaml.v3"
	"reflect"
	"strings"
)

type ConfigExporter interface {
	GetConfig(mode mode.Mode) properties.Properties
	GetConfigWraps() []*ConfigWrap
}

const (
	Append                   = mode.M1
	OnlyNew                  = mode.M2
	AnnotationSource         = mode.M3
	AnnotationSourceProperty = mode.M4
	AnnotationArgs           = mode.M5
)

type postProcessor struct {
	processors.DefaultInstantiationAwareComponentPostProcessor
	definition.PriorityComponent
	configure  configure.Configure
	quoteEl    el.Helper
	exprEl     el.Helper
	wraps      []*ConfigWrap
	properties []*component_definition.Property
	//propertyOriginArgs map[string]component_definition.TagArg
}

func (d *postProcessor) PostProcessComponentFactory(factory factory.Factory) error {
	d.configure = factory.GetConfigure()
	return nil
}

func NewConfigExporter() ConfigExporter {
	return &postProcessor{
		quoteEl: el.NewQuote(),
		exprEl:  el.NewExpr(),
		//propertyOriginArgs: make(map[string]component_definition.TagArg),
	}
}

func (d *postProcessor) Order() int {
	return -1
}

type ConfigWrap struct {
	ComponentName string
	Property      *component_definition.Property
	Prefix        string
	RealValue     any
}

const (
	modifiedTag = "@modified="
)

func (d *postProcessor) PostProcessBeforeInstantiation(m *component_definition.Meta, componentName string) (any, error) {
	if _, ok := m.Raw.(*app.App); ok {
		return m.Raw, nil
	}
	for _, prop := range m.GetAllProperties() {
		//copyArg, ok := d.propertyOriginArgs[prop.ID()]
		//if !ok {
		//	copyArg = component_definition.TagArg{}
		//	d.propertyOriginArgs[prop.ID()] = copyArg
		//}
		//for argType, vals := range prop.Args() {
		//	copyArg[argType] = vals
		//}
		prop.SetArg(component_definition.ArgRequired, []string{modifiedTag + "true"})
		d.properties = append(d.properties, prop)
	}
	return nil, nil
}

func (d *postProcessor) PostProcessBeforeInitialization(component any, componentName string) (any, error) {
	return nil, nil
}

func (d *postProcessor) GetConfigWraps() []*ConfigWrap {
	return d.wraps
}

func holderName(s *component_definition.Holder) string {
	if s.IsEmbed {
		return fmt.Sprintf("%s.Embed(%s)", holderName(s.Holder), s.Type.Name())
	}
	return s.Meta.Name()
}

func (d *postProcessor) GetConfig(mode mode.Mode) properties.Properties {
	pm := properties.New()
	for _, property := range d.properties {
		for p, a := range property.Configurations {
			prefix := p
			value := a
			if value == nil {
				value = reflectx.ZeroValue(property.Type)
			}

			if mode.Eq(AnnotationArgs) {
				tagArg := property.Args()
				tagArg.ForEach(func(argType component_definition.ArgType, args []string) {
					var trimArgs []string
					for _, arg := range args {
						if strings.HasPrefix(arg, modifiedTag) {
							arg = arg[len(modifiedTag):]
						}
						trimArgs = append(trimArgs, arg)
					}
					property.SetArg(argType, trimArgs)
				})
				//tagArg := d.propertyOriginArgs[property.ID()]
				//for argType, vals := range d.propertyOriginArgs[property.ID()] {
				//	if _, ok := tagArg[argType]; ok {
				//		tagArg[argType] = vals
				//	}
				//}
				//tagArg := property.Args()
				tagArg.ForEach(func(argType component_definition.ArgType, args []string) {
					pm.Set(fmt.Sprintf("%s@Args.%s", prefix, argType), args)
				})
			}

			if mode.Eq(AnnotationSource | AnnotationSourceProperty) {
				source := holderName(property.Holder)
				if mode.Eq(AnnotationSourceProperty) {
					source = fmt.Sprintf("%s.Field(%s).Tag(%s:'%s').Type(%s)", source, property.StructField.Name, property.Tag, property.TagVal, property.PropertyType)
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
					continue
				}
				if mode.Eq(Append) {
					value = origin
				}
			}

			switch property.Type.Kind() {
			case reflect.Pointer, reflect.Struct, reflect.Map:
				subRaw := toMap(value)
				subProp := properties.NewFromMap(subRaw)
				for subP, subAnyVal := range subProp {
					pm.Set(prefix+"."+subP, subAnyVal)
				}
			default:
				pm.Set(prefix, value)
			}
		}
	}
	return pm
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
