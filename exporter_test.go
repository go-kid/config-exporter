package config_exporter

import (
	"fmt"
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
	S     string         `yaml:"s"`
	B     bool           `yaml:"b"`
	M     map[string]int `yaml:"m"`
	Slice []float64      `yaml:"slice"`
	Sub   SubConfig      `yaml:"sub"`
	SubP  *SubConfig     `yaml:"subP"`
}

func (c *MergeConfig) Prefix() string {
	return "Merge"
}

type MergeParent struct {
	S2     string            `prop:"Merge.s2"`
	B2     bool              `prop:"Merge.b2"`
	M2     map[string]string `prop:"Merge.m2"`
	Slice2 []int64           `prop:"Merge.slice2"`
	Sub2   SubConfig         `prop:"Merge.sub2"`
	SubP2  *SubConfig        `prop:"Merge.subP2"`
}

type A struct {
	MergeParent
	ConfigA     string   `prop:"app.configA"`
	ConfigB     string   `prop:"${app.configB:config}"`
	ConfigSlice []string `value:"${app.configSlice:[a,b]}"`
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
        string: string
    s: string
    s2: string
    slice:
        - 0
    slice2:
        - 0
    sub:
        sub: string
    sub2:
        sub: string
    subP:
        sub: string
    subP2:
        sub: string
app:
    configA: string
    configB: config
    configSlice:
        - a
        - b
    valueB: abc
config: string
`)

func TestConfigExporter(t *testing.T) {
	t.Run("DefaultMode", func(t *testing.T) {
		a := &A{}
		exporter := NewConfigExporter()
		_, err := ioc.Run(
			app.LogTrace,
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
    configB: cfgB
    configSlice:
        - a
        - b
    valueB: abc
`)
		a := &A{}
		exporter := NewConfigExporter()
		_, err := ioc.Run(
			app.LogWarn,
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
        string: string
    s: string
    s2: string
    slice:
        - 0
    slice2:
        - 0
    sub:
        sub: string
    sub2:
        sub: string
    subP:
        sub: string
    subP2:
        sub: string
app:
    configA: cfgA
    configB: cfgB
    configSlice:
        - a
        - b
    valueB: abc
cfgB: string
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
			app.LogWarn,
			app.AddConfigLoader(loader.NewRawLoader(cfg)),
			app.SetComponents(a, exporter),
		)
		assert.NoError(t, err)
		bytes, err := yaml.Marshal(exporter.GetConfig(OnlyNew).Expand())
		assert.NoError(t, err)

		var exampleConfig = []byte(`Merge:
    b2: false
    m2:
        string: string
    s2: string
    slice2:
        - 0
    sub2:
        sub: string
    subP2:
        sub: string
app:
    configA: string
    configB: config
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
			app.LogWarn,
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
config@Sources:
    - github.com/go-kid/config-exporter/A
`)
		assert.Equal(t, string(exampleConfig), string(bytes))

		bytes, err = yaml.Marshal(exporter.GetConfig(AnnotationSourceProperty | OnlyNew).Expand())
		assert.NoError(t, err)
		exampleConfig = []byte(`Demo@Sources:
    - github.com/go-kid/config-exporter/A.Field(Config).Tag(prop:'Demo').Type(configuration)
    - github.com/go-kid/config-exporter/A2.Field(Config).Tag(prop:'Demo').Type(configuration)
Merge:
    b2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(B2).Tag(prop:'Merge.b2').Type(configuration)
    m2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(M2).Tag(prop:'Merge.m2').Type(configuration)
    s2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(S2).Tag(prop:'Merge.s2').Type(configuration)
    slice2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(Slice2).Tag(prop:'Merge.slice2').Type(configuration)
    sub2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(Sub2).Tag(prop:'Merge.sub2').Type(configuration)
    subP2@Sources:
        - github.com/go-kid/config-exporter/A.Embed(MergeParent).Field(SubP2).Tag(prop:'Merge.subP2').Type(configuration)
Merge@Sources:
    - github.com/go-kid/config-exporter/A.Field(Merge).Tag(prop:'Merge').Type(configuration)
    - github.com/go-kid/config-exporter/A2.Field(MergeConfig).Tag(prop:'Merge').Type(configuration)
app:
    configA@Sources:
        - github.com/go-kid/config-exporter/A.Field(ConfigA).Tag(prop:'app.configA').Type(configuration)
    configB@Sources:
        - github.com/go-kid/config-exporter/A.Field(ConfigB).Tag(prop:'config').Type(configuration)
    configSlice@Sources:
        - github.com/go-kid/config-exporter/A.Field(ConfigSlice).Tag(value:'["a","b"]').Type(configuration)
    valueB@Sources:
        - github.com/go-kid/config-exporter/A.Field(ValueB).Tag(value:'abc').Type(configuration)
config@Sources:
    - github.com/go-kid/config-exporter/A.Field(ConfigB).Tag(prop:'config').Type(configuration)
`)
		assert.Equal(t, string(exampleConfig), string(bytes))
	})
	t.Run("AnnotationArgsMode", func(t *testing.T) {
		exporter := NewConfigExporter()
		_, err := ioc.Run(
			app.LogWarn,
			app.SetComponents(&A{}, exporter),
			app.AddConfigLoader(loader.NewRawLoader(defaultConfig)),
		)
		assert.NoError(t, err)
		bytes, err := yaml.Marshal(exporter.GetConfig(AnnotationArgs | OnlyNew).Expand())
		assert.NoError(t, err)
		var exampleConfig = []byte(`Demo@Args:
    required:
        - "true"
Merge:
    b2@Args:
        required:
            - "true"
    m2@Args:
        required:
            - "true"
    s2@Args:
        required:
            - "true"
    slice2@Args:
        required:
            - "true"
    sub2@Args:
        required:
            - "true"
    subP2@Args:
        required:
            - "true"
Merge@Args:
    required:
        - "true"
app:
    configA@Args:
        required:
            - "true"
    configB@Args:
        required:
            - "true"
    configSlice@Args:
        required:
            - "true"
    valueB@Args:
        required:
            - "true"
config@Args:
    required:
        - "true"
`)
		assert.Equal(t, string(exampleConfig), string(bytes))
		fmt.Println(string(bytes))
	})
}
