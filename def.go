package config_exporter

import (
	"github.com/go-kid/ioc/component_definition"
	"github.com/go-kid/ioc/util/mode"
	"github.com/go-kid/ioc/util/properties"
)

type ConfigExporter interface {
	GetConfig(mode mode.Mode) properties.Properties
	ForEachConfiguration(f Iterator)
}

type Iterator func(property *component_definition.Property, prefix string, val any)

const (
	OnlyNew                  = mode.M2
	AnnotationSource         = mode.M3
	AnnotationSourceProperty = mode.M4
	AnnotationArgs           = mode.M5
)
