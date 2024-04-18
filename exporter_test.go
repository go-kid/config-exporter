package config_exporter

import (
	"github.com/go-kid/ioc"
	"github.com/go-kid/ioc/app"
	"github.com/go-kid/ioc/configure/loader"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

type SubConfig struct {
	Sub string `yaml:"sub"`
}

type Config struct {
	A     string         `yaml:"A"`
	B     int            `yaml:"B"`
	Slice []string       `yaml:"Slice"`
	Array [3]float64     `yaml:"Array"`
	M     map[string]int `yaml:"M"`
	G     Greeting       `yaml:"-"`
}

func (c *Config) Prefix() string {
	return "Demo"
}

type MergeConfig struct {
	S     string         `yaml:"S" json:"S"`
	B     bool           `yaml:"B" json:"B"`
	M     map[string]int `yaml:"M" json:"M"`
	Slice []float64      `yaml:"Slice" json:"Slice"`
	Sub   SubConfig      `yaml:"Sub" json:"Sub"`
	SubP  *SubConfig     `yaml:"SubP" json:"SubP"`
}

func (c *MergeConfig) Prefix() string {
	return "Merge,mapper=yaml"
}

type MergeParent struct {
	S2     string            `prop:"Merge.S2:s2"`
	B2     bool              `prop:"Merge.B2"`
	M2     map[string]string `prop:"Merge.M2:map[foo:bar]"`
	Slice2 []int64           `prop:"Merge.Slice2:[1,2,3]"`
	Sub2   SubConfig         `prop:"Merge.Sub2"`
	SubP2  *SubConfig        `prop:"Merge.SubP2:map[sub:sub]"`
}

type PartialZeroValue struct {
	Sub1 *SubConfig `yaml:"Sub1"`
	Sub2 *SubConfig `yaml:"Sub2"`
	Sub3 *SubConfig `yaml:"Sub3"`
	Sub4 *SubConfig `yaml:"Sub4"`
}

type A struct {
	MergeParent
	ConfigA          string   `prop:"app.configA"`
	ConfigB          string   `prop:"app.configB:b,validate=eq=b"`
	ConfigSlice      []string `value:"${app.configSlice:[a,b]},validate=min=1 max=10 required"`
	ValueA           string   `value:"abc"`
	ValueB           string   `value:"${app.valueB:abc}"`
	ValueC           string   `value:"#{'a'+'b'}"`
	Config           *Config
	Merge            *MergeConfig
	Greeting         Greeting              `wire:""`
	PartialZeroValue *PartialZeroValue     `prefix:"PartialZeroValue"`
	PartialZeroMap   map[string]*SubConfig `prop:"PartialZeroMap:map[sub2:map[sub:sub2]]"`
}

func (a *A) Init() error {
	return nil
}

func (a *A) Order() int {
	return 0
}

func (a *A) Run() error {
	a.Greeting.Hi()
	return nil
}

type Greeting interface {
	Hi()
}

var defaultConfig = []byte(`Demo:
    A: string
    Array:
        - 0
        - 0
        - 0
    B: 0
    M:
        string: 0
    Slice:
        - string
Merge:
    B: false
    B2: false
    M:
        string: 0
    M2:
        foo: bar
    S: string
    S2: s2
    Slice:
        - 0
    Slice2:
        - 1
        - 2
        - 3
    Sub:
        sub: string
    Sub2:
        sub: string
    SubP:
        sub: string
    SubP2:
        sub: sub
PartialZeroMap:
    sub2:
        sub: sub2
PartialZeroValue:
    Sub1:
        sub: string
    Sub2:
        sub: string
    Sub3:
        sub: string
    Sub4:
        sub: string
app:
    configA: string
    configB: b
    configSlice:
        - a
        - b
    valueB: abc
`)

func TestConfigExporter(t *testing.T) {
	t.Run("DefaultMode", func(t *testing.T) {
		a := &A{}
		exporter := NewConfigExporter()
		_, err := ioc.Run(
			app.LogError,
			app.SetComponents(a, exporter),
		)
		assert.NoError(t, err)
		bytes, err := yaml.Marshal(exporter.GetConfig(0))
		assert.NoError(t, err)

		assert.Equal(t, string(defaultConfig), string(bytes), string(bytes))
	})
	cfg := []byte(`
Demo:
    A: this is a test
    B: 20
    Slice:
        - "hello"
        - "world"
    Array:
        - 999
        - 888
        - 777
    M:
        Select: 22
Merge:
    B: true
    M:
        Select: 33
    S: "hello"
    Slice:
        - 9
        - 8
    Sub:
        sub: sub sub
    SubP:
        sub: subP sub
app:
    configA: cfgA
    configB: b
    configSlice:
        - a
        - b
    valueB: abc
PartialZeroValue:
    Sub1:
        sub: sub1
    Sub3:
        sub: sub3
PartialZeroMap:
    Sub1:
        sub: sub1
    Sub3:
        sub: sub3
`)
	t.Run("AppendMode", func(t *testing.T) {
		a := &A{}
		exporter := NewConfigExporter()
		_, err := ioc.Run(
			app.LogError,
			app.AddConfigLoader(loader.NewRawLoader(cfg)),
			app.SetComponents(a, exporter),
		)
		assert.NoError(t, err)
		bytes, err := yaml.Marshal(exporter.GetConfig(0))
		assert.NoError(t, err)

		var exampleConfig = []byte(`Demo:
    A: this is a test
    Array:
        - 999
        - 888
        - 777
    B: 20
    M:
        select: 22
    Slice:
        - hello
        - world
Merge:
    B: true
    B2: false
    M:
        select: 33
    M2:
        foo: bar
    S: hello
    S2: s2
    Slice:
        - 9
        - 8
    Slice2:
        - 1
        - 2
        - 3
    Sub:
        sub: sub sub
    Sub2:
        sub: string
    SubP:
        sub: subP sub
    SubP2:
        sub: sub
PartialZeroMap:
    sub1:
        sub: sub1
    sub3:
        sub: sub3
PartialZeroValue:
    Sub1:
        sub: sub1
    Sub2:
        sub: string
    Sub3:
        sub: sub3
    Sub4:
        sub: string
app:
    configA: cfgA
    configB: b
    configSlice:
        - a
        - b
    valueB: abc
`)
		assert.Equal(t, string(exampleConfig), string(bytes), string(bytes))
	})
	t.Run("OnlyNewMode", func(t *testing.T) {
		a := &A{}
		exporter := NewConfigExporter()
		_, err := ioc.Run(
			app.LogError,
			app.AddConfigLoader(loader.NewRawLoader(cfg)),
			app.SetComponents(a, exporter),
		)
		assert.NoError(t, err)
		bytes, err := yaml.Marshal(exporter.GetConfig(OnlyNew))
		assert.NoError(t, err)

		var exampleConfig = []byte(`Merge:
    B2: false
    M2:
        foo: bar
    S2: s2
    Slice2:
        - 1
        - 2
        - 3
    Sub2:
        sub: string
    SubP2:
        sub: sub
PartialZeroValue:
    Sub2:
        sub: string
    Sub4:
        sub: string
`)
		assert.Equal(t, string(exampleConfig), string(bytes), string(bytes))
	})
	t.Run("AnnotationSourceMode", func(t *testing.T) {
		type A2 struct {
			Config      *Config
			MergeConfig *MergeConfig
		}
		exporter := NewConfigExporter()
		_, err := ioc.Run(
			app.LogError,
			app.SetComponents(&A{}, &A2{}, exporter),
			app.AddConfigLoader(loader.NewRawLoader(defaultConfig)),
		)
		assert.NoError(t, err)
		bytes, err := yaml.Marshal(exporter.GetConfig(AnnotationSource | OnlyNew))
		assert.NoError(t, err)

		var exampleConfig = []byte(`Demo:
    A@Sources:
        - github.com/go-kid/config-exporter/A
        - github.com/go-kid/config-exporter/A2
    Array@Sources:
        - github.com/go-kid/config-exporter/A
        - github.com/go-kid/config-exporter/A2
    B@Sources:
        - github.com/go-kid/config-exporter/A
        - github.com/go-kid/config-exporter/A2
    M@Sources:
        - github.com/go-kid/config-exporter/A
        - github.com/go-kid/config-exporter/A2
    Slice@Sources:
        - github.com/go-kid/config-exporter/A
        - github.com/go-kid/config-exporter/A2
Merge:
    B@Sources:
        - github.com/go-kid/config-exporter/A
        - github.com/go-kid/config-exporter/A2
    B2@Sources: github.com/go-kid/config-exporter/A.Embed(MergeParent)
    M@Sources:
        - github.com/go-kid/config-exporter/A
        - github.com/go-kid/config-exporter/A2
    M2@Sources: github.com/go-kid/config-exporter/A.Embed(MergeParent)
    S@Sources:
        - github.com/go-kid/config-exporter/A
        - github.com/go-kid/config-exporter/A2
    S2@Sources: github.com/go-kid/config-exporter/A.Embed(MergeParent)
    Slice@Sources:
        - github.com/go-kid/config-exporter/A
        - github.com/go-kid/config-exporter/A2
    Slice2@Sources: github.com/go-kid/config-exporter/A.Embed(MergeParent)
    Sub:
        sub@Sources:
            - github.com/go-kid/config-exporter/A
            - github.com/go-kid/config-exporter/A2
    Sub2@Sources: github.com/go-kid/config-exporter/A.Embed(MergeParent)
    SubP:
        sub@Sources:
            - github.com/go-kid/config-exporter/A
            - github.com/go-kid/config-exporter/A2
    SubP2@Sources: github.com/go-kid/config-exporter/A.Embed(MergeParent)
PartialZeroMap@Sources: github.com/go-kid/config-exporter/A
PartialZeroValue:
    Sub1:
        sub@Sources: github.com/go-kid/config-exporter/A
    Sub2:
        sub@Sources: github.com/go-kid/config-exporter/A
    Sub3:
        sub@Sources: github.com/go-kid/config-exporter/A
    Sub4:
        sub@Sources: github.com/go-kid/config-exporter/A
app:
    configA@Sources: github.com/go-kid/config-exporter/A
    configB@Sources: github.com/go-kid/config-exporter/A
    configSlice@Sources: github.com/go-kid/config-exporter/A
    valueB@Sources: github.com/go-kid/config-exporter/A
`)
		assert.Equal(t, string(exampleConfig), string(bytes), string(bytes))
	})
	t.Run("AnnotationSourcePropertyMode", func(t *testing.T) {
		type A2 struct {
			Config      *Config
			MergeConfig *MergeConfig
		}
		exporter := NewConfigExporter()
		_, err := ioc.Run(
			app.LogError,
			app.SetComponents(&A{}, &A2{}, exporter),
			app.AddConfigLoader(loader.NewRawLoader(defaultConfig)),
		)
		assert.NoError(t, err)
		bytes, err := yaml.Marshal(exporter.GetConfig(AnnotationSource | OnlyNew))
		assert.NoError(t, err)

		bytes, err = yaml.Marshal(exporter.GetConfig(AnnotationSourceProperty | OnlyNew))
		assert.NoError(t, err)
		exampleConfig := []byte(`Demo:
    A@Sources:
        - github.com/go-kid/config-exporter/A.Field(Config).Type(Configuration).Tag(prefix:'Demo').TagActualValue(Demo).Required()
        - github.com/go-kid/config-exporter/A2.Field(Config).Type(Configuration).Tag(prefix:'Demo').TagActualValue(Demo).Required()
    Array@Sources:
        - github.com/go-kid/config-exporter/A.Field(Config).Type(Configuration).Tag(prefix:'Demo').TagActualValue(Demo).Required()
        - github.com/go-kid/config-exporter/A2.Field(Config).Type(Configuration).Tag(prefix:'Demo').TagActualValue(Demo).Required()
    B@Sources:
        - github.com/go-kid/config-exporter/A.Field(Config).Type(Configuration).Tag(prefix:'Demo').TagActualValue(Demo).Required()
        - github.com/go-kid/config-exporter/A2.Field(Config).Type(Configuration).Tag(prefix:'Demo').TagActualValue(Demo).Required()
    M@Sources:
        - github.com/go-kid/config-exporter/A.Field(Config).Type(Configuration).Tag(prefix:'Demo').TagActualValue(Demo).Required()
        - github.com/go-kid/config-exporter/A2.Field(Config).Type(Configuration).Tag(prefix:'Demo').TagActualValue(Demo).Required()
    Slice@Sources:
        - github.com/go-kid/config-exporter/A.Field(Config).Type(Configuration).Tag(prefix:'Demo').TagActualValue(Demo).Required()
        - github.com/go-kid/config-exporter/A2.Field(Config).Type(Configuration).Tag(prefix:'Demo').TagActualValue(Demo).Required()
Merge:
    B@Sources:
        - github.com/go-kid/config-exporter/A.Field(Merge).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
        - github.com/go-kid/config-exporter/A2.Field(MergeConfig).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
    B2@Sources: github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(B2).Type(Configuration).Tag(value:'${Merge.B2}').TagActualValue(false).Required()
    M@Sources:
        - github.com/go-kid/config-exporter/A.Field(Merge).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
        - github.com/go-kid/config-exporter/A2.Field(MergeConfig).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
    M2@Sources: github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(M2).Type(Configuration).Tag(value:'${Merge.M2:map[foo:bar]}').TagActualValue({"foo":"bar"}).Required()
    S@Sources:
        - github.com/go-kid/config-exporter/A.Field(Merge).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
        - github.com/go-kid/config-exporter/A2.Field(MergeConfig).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
    S2@Sources: github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(S2).Type(Configuration).Tag(value:'${Merge.S2:s2}').TagActualValue(s2).Required()
    Slice@Sources:
        - github.com/go-kid/config-exporter/A.Field(Merge).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
        - github.com/go-kid/config-exporter/A2.Field(MergeConfig).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
    Slice2@Sources: github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(Slice2).Type(Configuration).Tag(value:'${Merge.Slice2:[1,2,3]}').TagActualValue([1,2,3]).Required()
    Sub:
        sub@Sources:
            - github.com/go-kid/config-exporter/A.Field(Merge).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
            - github.com/go-kid/config-exporter/A2.Field(MergeConfig).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
    Sub2@Sources: github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(Sub2).Type(Configuration).Tag(value:'${Merge.Sub2}').TagActualValue({"sub":"string"}).Required()
    SubP:
        sub@Sources:
            - github.com/go-kid/config-exporter/A.Field(Merge).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
            - github.com/go-kid/config-exporter/A2.Field(MergeConfig).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
    SubP2@Sources: github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(SubP2).Type(Configuration).Tag(value:'${Merge.SubP2:map[sub:sub]}').TagActualValue({"sub":"sub"}).Required()
PartialZeroMap@Sources: github.com/go-kid/config-exporter/A.Field(PartialZeroMap).Type(Configuration).Tag(value:'${PartialZeroMap:map[sub2:map[sub:sub2]]}').TagActualValue({"sub2":{"sub":"sub2"}}).Required()
PartialZeroValue:
    Sub1:
        sub@Sources: github.com/go-kid/config-exporter/A.Field(PartialZeroValue).Type(Configuration).Tag(prefix:'PartialZeroValue').TagActualValue(PartialZeroValue).Required()
    Sub2:
        sub@Sources: github.com/go-kid/config-exporter/A.Field(PartialZeroValue).Type(Configuration).Tag(prefix:'PartialZeroValue').TagActualValue(PartialZeroValue).Required()
    Sub3:
        sub@Sources: github.com/go-kid/config-exporter/A.Field(PartialZeroValue).Type(Configuration).Tag(prefix:'PartialZeroValue').TagActualValue(PartialZeroValue).Required()
    Sub4:
        sub@Sources: github.com/go-kid/config-exporter/A.Field(PartialZeroValue).Type(Configuration).Tag(prefix:'PartialZeroValue').TagActualValue(PartialZeroValue).Required()
app:
    configA@Sources: github.com/go-kid/config-exporter/A.Field(ConfigA).Type(Configuration).Tag(value:'${app.configA}').TagActualValue(string).Required()
    configB@Sources: github.com/go-kid/config-exporter/A.Field(ConfigB).Type(Configuration).Tag(value:'${app.configB:b}').TagActualValue(b).Required().Validate(eq=b)
    configSlice@Sources: github.com/go-kid/config-exporter/A.Field(ConfigSlice).Type(Configuration).Tag(value:'${app.configSlice:[a,b]}').TagActualValue(["a","b"]).Required().Validate(min=1,max=10,required)
    valueB@Sources: github.com/go-kid/config-exporter/A.Field(ValueB).Type(Configuration).Tag(value:'${app.valueB:abc}').TagActualValue(abc).Required()
`)
		assert.Equal(t, string(exampleConfig), string(bytes), string(bytes))
	})
	t.Run("AnnotationArgsMode", func(t *testing.T) {
		exporter := NewConfigExporter()
		_, err := ioc.Run(
			app.LogError,
			app.SetComponents(&A{}, exporter),
			app.AddConfigLoader(loader.NewRawLoader(defaultConfig)),
		)
		assert.NoError(t, err)
		bytes, err := yaml.Marshal(exporter.GetConfig(AnnotationArgs | OnlyNew))
		assert.NoError(t, err)
		var exampleConfig = []byte(`Demo@Args:
    Required: true
Merge:
    B2@Args:
        Required: true
    M2@Args:
        Required: true
    S2@Args:
        Required: true
    Slice2@Args:
        Required: true
    Sub2@Args:
        Required: true
    SubP2@Args:
        Required: true
Merge@Args:
    Mapper:
        - yaml
    Required: true
PartialZeroMap@Args:
    Required: true
PartialZeroValue@Args:
    Required: true
app:
    configA@Args:
        Required: true
    configB@Args:
        Required: true
        Validate:
            - eq=b
    configSlice@Args:
        Required: true
        Validate:
            - min=1
            - max=10
            - required
    valueB@Args:
        Required: true
`)
		assert.Equal(t, string(exampleConfig), string(bytes), string(bytes))
	})
}
