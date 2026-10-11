# Web V12 → 游戏：可测量差异清单

2026-10-11。本轮停止抽象评分。**此前91分不能作为“完全复刻Web”的验收结论；功能存在不等于视觉一致。** 以Web最终computed style及相同viewport实测矩形为准，不再用“接近”“基本恢复”代替对照。

原始完整记录：`output/web-v12-computed-exact.json`。本次独立浏览器 DPR=1，分别1440×900、1920×1080，加载 `?v=12&gui=3`，字体ready后打开完整聊天，等待300ms。最终CSS是 preview.css 后加载 hud-reference.css，历史规则需按级联覆盖。消息滚动到末尾时第一条DOM矩形y可能为负；**使用其尺寸，不能照搬该y坐标**。

## 1. 页面骨架（物理px）

矩形格式 x,y,w,h，测量小数可按设备像素栅格统一舍入。

| 区域 | 1440×900 | 1920×1080 |
| --- | --- | --- |
| topbar | 0,0,1440,76 | 0,0,1920,88 |
| brand整体 | 32,27.5,134.92,21 | 40,33.5,134.92,21 |
| rail | 0,76,104,792 | 0,88,116,952 |
| 首频道按钮 | 18,76,74,73 | 24,88,80,82 |
| 频道组 | 18,76,74,310 | 24,88,80,346 |
| 频道→联系人分隔线 | 32,405,46,1 | 38,453,52,1 |
| 联系人容器 | 18,418,74,394 | 24,466,80,518 |
| content外框 | 136,76,662.39,792 | 158,88,870,952 |
| heading实际可用宽 | 136,76,643.39,59 | 158,88,851,66 |
| 标题文字框 | 138,83,639.39,30 | 160,96.70,847,33.59 |
| messages视口 | 136,135,643.39,627 | 158,154,851,771 |
| composer整体 | 136,762,643.39,106 | 158,925,851,115 |
| tools行 | 136,768,643.39,43 | 158,931,851,43 |
| 第一个工具按钮 | 136,771.5,36,36 | 158,934.5,36,36 |
| 输入行 | 136,819,643.39,49 | 158,986,851,54 |
| 输入框 | 136,819,464.39,49 | 158,986,672,54 |
| 发送按钮逻辑框 | 615.39,821.5,164,44 | 845,991,164,44 |

**完整页应先按物理viewport计算Web断点和布局，再除Minecraft GUI scale交给LDLib2。** 不能把固定45逻辑单位等同Web的136px：GUI3碰巧近似，GUI2就会缩成90px。HUD则保留已确认的GUI×0.6逻辑；两套尺度不得混用。

当前ChatView源码可直接确认的偏差：content外框宽约正确但heading/timeline/input复用全部宽度，少扣Web右padding19px；GUI3 heading当前x135，Web136；频道y81而Web76；联系人整体y423而Web418。请修整套布局公式，不逐个加补丁偏移。

## 2. Sidebar联系人（重点）

| 项目 | Web最终值 |
| --- | --- |
| 单联系人框 | 1440下74×68px；1920下80×76px；flex-column、水平居中、内容垂直居中 |
| 头像 | 两断点均30×30px，不随1920加大；r3；背景#292E30；外框1px rgba(217,225,210,.137)；nearest |
| 姓名 | **Gilroy Medium500，11px，line-height14.3px，letter-spacing normal，无阴影**；中文回退Harmony。最新颜色统一粉色，但字体不变 |
| 头像与姓名 | gap6px；不能固定左对齐文本 |
| 未读 | 最小17×17px、r10、水平padding4；黄#FFF100；字#242826；Novecento500 10px/10px；外暗圈2px #171D20；按钮右4px、上4px |
| 首联系人头像1440 | x40,y426.84,w30,h30 |
| 首联系人姓名Amiya1440 | x39.36,y462.84,w31.27,h14.30 |
| 首未读1440 | x70,y423,w17,h17 |
| 首头像/姓名/未读1920 | (49,478.84,30,30)/(48.36,514.84,31.27,14.30)/(82,471,17,17) |

当前游戏refreshContacts使用ChatStyle.label默认Harmony、size4×GUI3≈12px，并在局部x2,y12的固定宽标签中画名字；Web是Gilroy11px居中。**这是ID字形/尺寸/横向位置都不一致的直接原因，不能只调字号。** 当前头像local x9×GUI3导致x45，Webx40；badge18px而Web17；行步距66而Web68。缺失头像r3/细边、昵称省略号规则也需补。

行为附带差异：Web sidebar仅pinned列表；当前游戏遍历orderedContacts前7条而未筛pinned，且硬截7条，和Web可滚动联系人容器不同。这个差异直接影响侧栏布局验收。

## 3. 字体与消息

| 元素 | 1440×900 | ≥1800宽 |
| --- | --- | --- |
| 标题 | Harmony500 25px/30px，tracking1px | 28px/33.6px |
| 消息ID | Gilroy500 13px/16.9px | 14px/18.2px |
| 头衔 | Harmony400 11px/14.3px | 12px/15.6px |
| 时间数字 | Novecento500 10px/10px，#9DA8AB | 11px/11px |
| 正文 | Harmony400 15px/22.5px，tracking.12px | 16px/24px |
| 消息头像 | 32px，r3 | 35px，r3 |
| 头像到正文列 | 12px | 14px |
| 条间距 | 8px | 9px |
| 正文padding | 4px 11px 4px 13px | 4px 12px 4px 14px |
| 元信息→正文 | 3px | 3px |

元信息gap12px；名字串内部原gap7px。姓名/头衔/位置順序按最新业务要求替换，不复制旧Web的随机头衔和颜色。完整页数字时间不得继续用HarmonyRegular代替Novecento；“刚刚”须中文回退字体。

完整页气泡：**同一原纹理** bg_snscontenttextorpic_chat_03.png，border-image-slice31 31 38 38 fill，实际边8/10/10/12px，alpha.055（hover.095），内容本身transparent。radius0不代表可画硬矩形；轮廓由纹理透明区域决定。

品牌不是一个字符串同字体：Deuterium=Geist650 21px/21px、tracking−.7px；间隔7px；IX=BrandIX700 20px；其后3×3黄色点，margin-left5px。当前单Label整串品牌不是精确复刻。

字体验收不能只看provider声明size9：用相同字符串截图校验基线、实际墨迹高与宽。Web侧栏Amiya文本框宽31.27px（11pxGilroy500）可作最小校准样本；Minecraft provider oversample、LDLib文本scale及GUI会共同影响最终结果。

## 4. 背景三层：不能用加黑冒充磨砂

按从世界到UI的顺序分别合成，不能把三层压成单一黑色mask。

### A. 磨砂层

`backdrop-filter:blur(14px)`，物理像素半径/视觉模糊宽度，不是14GUI单位。mask水平方向：0%→1，24%→1，42%→0.7333，58%→0.2667，76%→0；各段线性插值。1920宽对应x=0/460.8/806.4/1113.6/1459.2；1440宽对应0/345.6/604.8/835.2/1094.4。

需要同一背景纹理的对照，检查竖直高对比边缘扩散；仅“使用Gaussian”不能证明与14px一致。当前游戏不同世界截图只能定性看模糊，不能据此宣布完全相同。

### B. 暗色遮罩层

| 横向stop | RGB | alpha |
| --- | --- | --- |
| 0% | 16,20,23 | 252/255 |
| 18% | 17,22,25 | 244/255 |
| 34% | 19,25,27 | 232/255 |
| 48% | 19,25,27 | 187/255 |
| 61% | 20,26,29 | 104/255 |
| 75% | 20,26,29 | 28/255 |
| 90% | transparent | 0 |

不仅alpha渐变，**RGB也分段变化**。已读游戏shader仍用固定tint vec3(.063,.078,.09)，这不是精确匹配。颜色插值应遵循预乘alpha合成，避免淡出边界色偏。

### C. 散点/点阵层（当前缺失的质感）

- 原图：assets/controls/deco_lattice_title.png。
- repeat平铺，背景tile为 **142×66物理px**，锚定屏幕(0,0)，不是世界坐标也不是GUI缩放tile。
- 整层opacity **.022**。
- mask `linear-gradient(90deg, #000, transparent 65%)`：左端1，x=.65W降为0（1440为936px，1920为1248px）。
- 放在暗色/磨砂之后、UI文字之前；勿用随机噪声替代原图。已确认游戏资源虽有deco_lattice_title.png，但当前Java/shader未查到绘制引用，**文件存在不等于点阵已渲染**。

## 5. Composer和状态

tools按钮36px，原SNS实体表情图形24px；不是线性笑脸替代。tools与input gap8px（1920为12px），input-row gap15px。draft字体1440为Harmony15px，1920为16px；可见边1px，不随GUI变为2/3px。

发送按钮最终外框164×44px，两断点不随GUI放大；纹理层有内部inset与溢出，不能把整个native sprite直接等比压成逻辑框；以固定图层九切及hover/pressed精确截图对照。频道状态同样要按左线/菱形参数，不仅颜色一致。

## 6. 后续验收方式（不评分）

1. 固定1440×900、DPR对应真实帧缓冲、GUI3，将上表关键rect数值打印到测试报告；GUI2再测相同完整页物理rect，应保持Web目标，不整体缩水。HUD仍按独立GUI规则。
2. 固定相同联系人顺序/同一句文字/同一条未读，截sidebar 104px区域与composer区域，比较ID文字宽、baseline、头像位置。
3. 背景同材质测试图下对比14px边缘扩散、各stop采样值、点阵142×66重复周期；否则不得宣称模糊完全相同。
4. 完整页、侧栏、个人页、查看器逐区域对齐后再验交互。此前UI功能通过保留为功能证据，不能抵消任何视觉偏差。

## 7. 底部按钮、通知圆点与音效：最新只读检查

本节基于新增WebPixels/RoundTexture/ChatSounds代码，尚无这一版按钮状态实机图，不认定AA和中心位置已视觉验收。

### 已确认的实现与仍缺差异

- `harmony.ttf` 实际OS/2字重500、子族Medium，与Web HarmonyOS_Sans_SC_Medium.ttf一致；发送文案应保持 **Harmony500 15px/15px**，不换成只用于IX字标的Brand字体。当前send字号5×3=15方向正确，但需测字墨迹基线/宽度，而不是仅看fontSize。
- NativeButtonTexture当前画normal-bg与hover-bg两层；Web实际还包含 **native-contour（alpha .12，inset7px 14px）**、hover才显的细outline、**右侧16×16中性灰pointer（right−22px，top居中）**。这些层尚未出现在该类绘制中，九切边界正确也不能称材质完整。pressed需要Web的暗化与1px下移。
- RoundTexture直alpha输出配defaultBlend方向正确，但必须用同尺寸17px图、非整数屏幕位置验证边缘覆盖，不能只因换shader判定抗锯齿完成。
- Web未读有额外 **2px #171D20外暗圈**，当前unread只RoundTexture填色未见外圈；字号Novecento500 10px，数字需按实际glyph baseline光学校准。补1/8/10/99样本，Web是min-width17、padding左右4，可随多位数增宽；不要所有数字硬塞固定17px。
- 工具图形已改原emoji/plus素材是正确方向。当前toolButton只有pressed黄态，没有对应“面板保持打开”的selected持久态；Web选中表情在打开期间保持黄色圆框，分享加号打开变45°关闭态。须将state绑定panel，而非鼠标按住才改变。
- 分享白圆属于用户最新增补，与历史V12末尾CSS的暗色覆写有区别，应明确作为用户新指令保留；不要再被早期CSS覆盖来回改。

### 音效精确路由（不能只核对六个文件存在）

Web幅度：open .22、click .18、send .6、card .3、panel .3、close .12；当前ChatSounds相同。没有独立听感试听，未确认ogg转码后的响度/瞬态一致。

源码可见的路由差异：

1. 当前所有ChatButton默认click，Web通用button工厂本身不播音，仅具体动作显式play。分享内部二级选项/返回、收藏等不能一律click；应按Web对应动作card路由。
2. MediaViewer打开有card，但从查看器关闭新建ChatScreenFull会播open；Web关闭大图恢复原页**不重新播整页open**。需区分首次完整页打开与viewer返回重建。
3. StickerTile短点击固定先card再send，适用于Web内置贴纸sendSticker路径；个人贴纸Web sendMedia只send。管理删除目前也复用StickerTile短点击，会错误播send，删除不是发送。
4. mute按钮默认click在切换前播放；Web关闭音效后play被抑制。要去掉该按钮默认role再按切换后状态处理，否则“关闭音效”仍响一声。
5. 当前按钮动作从mouseDown触发，Web从click/鼠标释放触发。若追求完全一致，确认按下拖离取消与声音触发时点；不能用role去重30ms掩盖不同动作重复发声。

建议通过事件记录列“动作→role序列”，逐条对比，不只测六种声音各播放一次。关闭弹层、删除、无效输入及disabled按钮也要有明确“不应播放send”的检查。

## 8. 21:10 实际截图收尾清单（按明显程度，不评分）

本次真实查看14_full-chat、18_composer-hover、26_stickers、31_personal-stickers、37_viewer、53_private-inbox、62_private-selection。三工具图标最新要求为 **36px命中、可见轮廓统一24px**（图片24×21.6，分享圆24、plus14），覆盖旧Web分享36圆，不再建议改回。

### 高优先：仍可直接看出的不一致

1. **发送hover箭头依然黄色。** 18_composer-hover的右箭头肉眼为黄；Web已确认中性灰`#c5cbc8`。若原图RGB为黄，乘灰色tint不会去色，需用alpha-mask填灰或等效去饱和；以截图结果验，不以变量名“gray”验。
2. **表情面板的分类结构尚未复制。** Web有个人/贴纸/常用/冒险；当前终末地/个人两项，底部四小符号不是常用/冒险的完整分类与网格。Web选择为短黄下划线和选中黄色文案，当前宽白线/暗色tab块，区别明显。
3. **个人库布局不同。** Web数量和管理在分类下的工具行，添加在网格末尾；当前管理与分类同排、数量沉到面板底、添加为右上小加号。应按Web层级摆放，不只做到能管理。
4. **私聊列表缺卡片材质与尺寸。** Web1440行最小84px、头像44px，淡底`#b4c0c009`、细边`#c5cfcd22`、右侧mail-strip纹理alpha.16；当前只有横线和36px头像，整页显得像表格。不是微小抗锯齿差异。
5. **批量选择形态不同。** Web复选框+选中弱黄边/淡底，底部应用/取消；当前整行灰块加顶部按钮，无checkbox。既然目标完全复刻，应复用Web选择层级，不以功能能多选代替。

### 中优先：查看器与整页出口

6. Web查看器footer缩放组靠右（33px按钮、16px间隔），header有底分隔、收藏矩形细边按钮；当前footer居中、收藏裸文字。顶标题Web显示媒体名，当前只“贴纸”。图片尺寸已修，不需要再次改fit来解决这些结构差异。
7. 完整页右上仍普通×；Web是原close-sprite加Esc键帽。最新删的是屏幕Enter完整聊天提示，不包括这个Esc键帽，不能把两项混淆。
8. 面板打开时emoji图形仍灰色，仅外环黄；Webselected图形本身也黄色。应将持久selected颜色应用到图形素材。

### 不再列为主要阻断差异

- 主页面物理尺度、联系人位置和消息气泡已明显收敛，当前不要再大幅重排主区。
- 点阵已出现在最新截图；同背景磨砂实验的边缘误差最低值位于Web14px，详见 `output/same-background-blur-review.md`。**当前没有依据继续盲增blur**。
- 三工具可见尺寸依最新24px要求，避免因旧CSS反复来回修改。
- 声音改动仍需录制/事件序列验证；静态截图不能确认播放次数或音量，不在本节擅自判通过。

## 9. 21:23轮复核：仅列明显错误与具体交互差异

实际查看新发送hover、4类表情、个人库、viewer、分享/位置、私聊/选择截图。此前右箭头已为灰色，4tabs与工具行/网格加号、卡片/checkbox/底部应用、viewer右侧控件/分隔线、原关闭+Esc均在截图中得到修正。不恢复用户明确覆盖的toolbar可见24px要求。

发现一处明显回归：63_share-panel当时变成浅色440px面板+蓝位置图块，这是旧CSS规则；Web末尾有效规则为深色354px、65px图块。已即时反馈主线，主线随后报告改为正确深色规则。须以新截图确认，不把toolbar白圆推导成面板也应浅色。

### Emoji实际剩余差异（源码可确认，截图尚未发消息覆盖）

1. **插入位置错误。** ChatPanels常用/冒险按钮以`input.getValue()+value`拼接，永远追加末尾；Web是`setRangeText`按当前光标/选区插入替换。复现步骤：输入AB，将光标放A与B之间，选😀，应为A😀B；选中B再选😀应替换B。
2. **纯emoji消息样式漏分支。** Web通过Extended_Pictographic判断emoji-only，字号29px、无气泡底、padding0；ChatTimeline当前所有文字仍走普通正文气泡。需真实发😀与A😀B比较，只有纯emoji采用特殊显示，混排不应改变正文行高到异常。

输入和正文的harmony/harmony_regular字体都已引用emoji位图provider，因此不再泛泛怀疑“发送后会方块”；必须以具体字符或实际截图说明缺字。当前检查没有证据认定其缺字。

本轮仍不评分，不用93步/3checks替代上述光标和纯emoji样例的实际验证。

## 10. 21:38 定向闭环复核

实际查看最新14_full-chat、55_common-emoji、68_share-panel、79_private-conversation，并读取run43516_23cfc9eb PASS报告（105步，4/4明确检查）及插入选区断言源码。

- **选区插入已闭环：** 场景真实click表情格后断言`ABC`选中B得到`A😀C`；68_share-panel输入框也可见A😀C，未追加到结尾。
- **彩色emoji与纯emoji已闭环：** 14_full-chat中“收到，出发！😀✨”混排显示彩色图形且保持正文行；xiwanzi的独立😀约29px、无气泡底。不是仅在选择面板显示彩色。
- **分享面板回归已闭环：** 68_share-panel已为深色354px，内部两项均为统一暗色图块；不再是旧浅色/蓝块方案。
- **私聊标题新样式可见：** 79_private-conversation包含返回、粉色Amiya、灰头衔、固定图标，消息后缀显示私聊；当前截图无标题截断或按钮压盖。
- 本轮限定范围内未发现新的明显视觉回归。保留用户最新工具图标24px要求，不再引用旧share尺寸。

本节是指定闭环点的证据确认，不评分、不扩展成真实后端/S3/App生产联调通过，也不宣称全部像素已无差异。GUI2第二次105步运行由主线报告，本评审未另外打开该次全套截图。
