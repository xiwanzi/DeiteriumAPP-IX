# 旅者之乡 2.2.8 远景雾来源核查

2026-10-13。历史会话「识别整合包原创模组」提供了本机路径。本次只读核查原始 `C:/Users/34545/Downloads/旅者之乡2.2.8.7z` 及解压目录，没有启动整合包或修改其中配置。

## 结论

不能把整合包默认画面的远景雾全部归给一个独立模组。原始分发包默认开启 **Iris + BSL v10.1 光影**；同时装有并启用 **Dynamic Surroundings（动态环绕）0.4.2** 的环境雾增强。具体所见效果取决于当时是否开着光影。

### 原包默认光影

从原始7z直接读取的 `config/iris.properties` 为 `enableShaders=true`、`shaderPack=BSL_v10.1.zip`。解压目录当前为 `enableShaders=false`，两者不同，因此不能只凭当前解压配置推断整合包默认画面。

BSL 的 `shaders/lib/atmospherics/fog.glsl` 中 `NormalFog` 按观察距离计算并混合雾色，包含昼夜、天气、高度及远端雾处理。设置入口定义在 `shaders/shaders.properties` 的 `screen.FOG`；`FOG_DENSITY`、`FOG_HEIGHT`、`FAR_VANILLA_FOG` 控制相关效果。原包的 BSL 设置文件没有覆盖这些雾参数。`MCBL_FOG=false` 对应另一项方块光照雾设置，不能理解为关闭全部距离雾。

### 动态环绕

实际 JAR 为 `【动态环绕】dynamicsurroundings-neoforge-1.21.1-0.4.2.jar`，模组 ID `dsurround`，作者 OreCruncher。原包和解压目录的 `config/dsurround/dsurround.json` 四项均为 `true`：`enableFogEffects`、`enableMorningFog`、`enableBiomeFog`、`enableWeatherFog`。

JAR 中 `FogHandler` 调用雾起止范围/形状设置，组合晨雾、群系雾及天气雾计算器。它是增强原版环境雾的实际模组；开启 BSL 时，最终远景还受光影自身雾计算影响。

### 其他核对

- 原始7z和当前模组目录均没有 Distant Horizons 主 JAR，仅有 `DistantHorizons.toml` 遗留配置，不能据此认定 DH 在提供雾效。
- Particle Rain 有雾粒子资源，但配置中的常驻 `mist` 与雨后 `mist_after_rain` 均关闭；资源名字存在不代表该效果启用。
- 距离雾是在已渲染画面的远端混入雾色、柔化边界，本身不会生成视距外的地形。
- 本轮只确认来源，没有把第三方光影或模组加入 Deuterium 客户端。

作者来源：[Dynamic Surroundings](https://github.com/OreCruncher/DynamicSurroundingsFabric)、[BSL Shaders](https://modrinth.com/shader/bsl-shaders)。具体版本与启用状态以以上本机原包核查为依据，网页当前版本不替代该分发包版本。
