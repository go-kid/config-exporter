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
	A     string         `yaml:"a"`
	B     int            `yaml:"b"`
	Slice []string       `yaml:"slice"`
	Array [3]float64     `yaml:"array"`
	M     map[string]int `yaml:"m"`
	G     Greeting       `yaml:"-"`
}

func (c *Config) Prefix() string {
	return "Demo"
}

type MergeConfig struct {
	S     string         `yaml:"s" json:"S"`
	B     bool           `yaml:"b" json:"B"`
	M     map[string]int `yaml:"m" json:"M"`
	Slice []float64      `yaml:"slice" json:"Slice"`
	Sub   SubConfig      `yaml:"sub" json:"Sub"`
	SubP  *SubConfig     `yaml:"subP" json:"SubP"`
}

func (c *MergeConfig) Prefix() string {
	return "Merge,mapper=yaml"
}

type MergeParent struct {
	S2     string            `prop:"Merge.s2:s2"`
	B2     bool              `prop:"Merge.b2"`
	M2     map[string]string `prop:"Merge.m2:map[foo:bar]"`
	Slice2 []int64           `prop:"Merge.slice2:[1,2,3]"`
	Sub2   SubConfig         `prop:"Merge.sub2"`
	SubP2  *SubConfig        `prop:"Merge.subP2:map[sub:sub]"`
}

type A struct {
	MergeParent
	ConfigA     string   `prop:"app.configA"`
	ConfigB     string   `prop:"app.configB:b,validate=eq=b"`
	ConfigSlice []string `value:"${app.configSlice:[a,b]},validate=min=1 max=10 required"`
	ValueA      string   `value:"abc"`
	ValueB      string   `value:"${app.valueB:abc}"`
	ValueC      string   `value:"#{'a'+'b'}"`
	Config      *Config
	Merge       *MergeConfig
	Greeting    Greeting `wire:""`
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
    a: string
    array:
        - 0
        - 0
        - 0
    b: 0
    m:
        string: 0
    slice:
        - string
Merge:
    b: false
    b2: false
    m:
        string: 0
    m2:
        foo: bar
    s: string
    s2: s2
    slice:
        - 0
    slice2:
        - 1
        - 2
        - 3
    sub:
        sub: string
    sub2:
        sub: string
    subP:
        sub: string
    subP2:
        sub: sub
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
		bytes, err := yaml.Marshal(exporter.GetConfig(0).Expand())
		assert.NoError(t, err)

		assert.Equal(t, string(defaultConfig), string(bytes))
	})

	t.Run("AppendMode", func(t *testing.T) {
		cfg := []byte(`
Demo:
    a: this is a test
    b: 20
    slice:
        - "hello"
        - "world"
    array:
        - 999
        - 888
        - 777
    m:
        select: 1
app:
    configA: cfgA
    configB: b
    configSlice:
        - a
        - b
    valueB: abc
`)
		a := &A{}
		exporter := NewConfigExporter()
		_, err := ioc.Run(
			app.LogError,
			app.AddConfigLoader(loader.NewRawLoader(cfg)),
			app.SetComponents(a, exporter),
		)
		assert.NoError(t, err)
		bytes, err := yaml.Marshal(exporter.GetConfig(Append).Expand())
		assert.NoError(t, err)

		var exampleConfig = []byte(`Demo:
    a: this is a test
    array:
        - 999
        - 888
        - 777
    b: 20
    m:
        select: 1
    slice:
        - hello
        - world
Merge:
    b: false
    b2: false
    m:
        string: 0
    m2:
        foo: bar
    s: string
    s2: s2
    slice:
        - 0
    slice2:
        - 1
        - 2
        - 3
    sub:
        sub: string
    sub2:
        sub: string
    subP:
        sub: string
    subP2:
        sub: sub
app:
    configA: cfgA
    configB: b
    configSlice:
        - a
        - b
    valueB: abc
`)
		assert.Equal(t, string(exampleConfig), string(bytes))
	})
	t.Run("OnlyNewMode", func(t *testing.T) {
		cfg := []byte(`Merge:
    b: false
    m:
        string: 0
    s: string
    slice:
        - 0
    sub:
        sub: "subSub"
        subP:
            sub: "subSubPSub"
    subP:
        sub: string
Demo:
    a: this is a test
    b: 20
    slice:
        - "hello"
        - "world"
    array:
        - 999
        - 888
        - 777
    m:
        select: 1
config: "hello"
`)
		a := &A{}
		exporter := NewConfigExporter()
		_, err := ioc.Run(
			app.LogError,
			app.AddConfigLoader(loader.NewRawLoader(cfg)),
			app.SetComponents(a, exporter),
		)
		assert.NoError(t, err)
		bytes, err := yaml.Marshal(exporter.GetConfig(OnlyNew).Expand())
		assert.NoError(t, err)

		var exampleConfig = []byte(`Merge:
    b2: false
    m2:
        foo: bar
    s2: s2
    slice2:
        - 1
        - 2
        - 3
    sub2:
        sub: string
    subP2:
        sub: sub
app:
    configA: string
    configB: b
    configSlice:
        - a
        - b
    valueB: abc
`)
		assert.Equal(t, string(exampleConfig), string(bytes))
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
		bytes, err := yaml.Marshal(exporter.GetConfig(AnnotationSource | OnlyNew).Expand())
		assert.NoError(t, err)

		var exampleConfig = []byte(`Demo@Sources:
    - github.com/go-kid/config-exporter/A
    - github.com/go-kid/config-exporter/A2
Merge:
    b2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent)
    m2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent)
    s2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent)
    slice2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent)
    sub2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent)
    subP2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent)
Merge@Sources:
    - github.com/go-kid/config-exporter/A
    - github.com/go-kid/config-exporter/A2
app:
    configA@Sources:
        - github.com/go-kid/config-exporter/A
    configB@Sources:
        - github.com/go-kid/config-exporter/A
    configSlice@Sources:
        - github.com/go-kid/config-exporter/A
    valueB@Sources:
        - github.com/go-kid/config-exporter/A
`)
		assert.Equal(t, string(exampleConfig), string(bytes))
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
		bytes, err := yaml.Marshal(exporter.GetConfig(AnnotationSource | OnlyNew).Expand())
		assert.NoError(t, err)

		bytes, err = yaml.Marshal(exporter.GetConfig(AnnotationSourceProperty | OnlyNew).Expand())
		assert.NoError(t, err)
		exampleConfig := []byte(`Demo@Sources:
    - github.com/go-kid/config-exporter/A.Field(Config).Type(Configuration).Tag(prefix:'Demo').TagActualValue(Demo).Required()
    - github.com/go-kid/config-exporter/A2.Field(Config).Type(Configuration).Tag(prefix:'Demo').TagActualValue(Demo).Required()
Merge:
    b2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(B2).Type(Configuration).Tag(value:'${Merge.b2}').TagActualValue(false).Required()
    m2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(M2).Type(Configuration).Tag(value:'${Merge.m2:map[foo:bar]}').TagActualValue({"foo":"bar"}).Required()
    s2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(S2).Type(Configuration).Tag(value:'${Merge.s2:s2}').TagActualValue(s2).Required()
    slice2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(Slice2).Type(Configuration).Tag(value:'${Merge.slice2:[1,2,3]}').TagActualValue([1,2,3]).Required()
    sub2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(Sub2).Type(Configuration).Tag(value:'${Merge.sub2}').TagActualValue({"sub":"string"}).Required()
    subP2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(SubP2).Type(Configuration).Tag(value:'${Merge.subP2:map[sub:sub]}').TagActualValue({"sub":"sub"}).Required()
Merge@Sources:
    - github.com/go-kid/config-exporter/A.Field(Merge).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
    - github.com/go-kid/config-exporter/A2.Field(MergeConfig).Type(Configuration).Tag(prefix:'Merge').TagActualValue(Merge).Mapper(yaml).Required()
app:
    configA@Sources:
        - github.com/go-kid/config-exporter/A.Field(ConfigA).Type(Configuration).Tag(value:'${app.configA}').TagActualValue(string).Required()
    configB@Sources:
        - github.com/go-kid/config-exporter/A.Field(ConfigB).Type(Configuration).Tag(value:'${app.configB:b}').TagActualValue(b).Required().Validate(eq=b)
    configSlice@Sources:
        - github.com/go-kid/config-exporter/A.Field(ConfigSlice).Type(Configuration).Tag(value:'${app.configSlice:[a,b]}').TagActualValue(["a","b"]).Required().Validate(min=1,max=10,required)
    valueB@Sources:
        - github.com/go-kid/config-exporter/A.Field(ValueB).Type(Configuration).Tag(value:'${app.valueB:abc}').TagActualValue(abc).Required()
`)
		assert.Equal(t, string(exampleConfig), string(bytes))
	})
	t.Run("AnnotationArgsMode", func(t *testing.T) {
		exporter := NewConfigExporter()
		_, err := ioc.Run(
			app.LogError,
			app.SetComponents(&A{}, exporter),
			app.AddConfigLoader(loader.NewRawLoader(defaultConfig)),
		)
		assert.NoError(t, err)
		bytes, err := yaml.Marshal(exporter.GetConfig(AnnotationArgs | OnlyNew).Expand())
		assert.NoError(t, err)
		var exampleConfig = []byte(`Demo@Args:
    Required: true
Merge:
    b2@Args:
        Required: true
    m2@Args:
        Required: true
    s2@Args:
        Required: true
    slice2@Args:
        Required: true
    sub2@Args:
        Required: true
    subP2@Args:
        Required: true
Merge@Args:
    Mapper:
        - yaml
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
