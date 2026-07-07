# 概述
产品名称：知玄
做一个ai助手类软件，基于 llm 成为用户的秘书和第二大脑

## 系统架构
前后端分离
前端用nodejs + vue 
后端用 go 
前后端代码放在同一个项目下，方便维护
部署形式：开发完成后需要上云，上云方式为在一台云服务器上部署前后端，使用nginx托管前端资源，转发接口请求到go进程
移动端适配，需要同时满足pc端和移动端的易用性

### 技术细节
开发环境机器：macbook
生产环境机器：阿里云服务器
开发阶段可以用 sqlite 做，同时需要兼容mysql，生成环境改用 mysql
做良好的后端架构，满足 go web 项目的最佳实践


## 一期需求(已完成)
### 需求目标
实现用户的注册和登录功能

### 后端
用户表设计
注册接口
登录接口，设置不失效cookie
统一接口登录状态校验

### 前端
注册页面
登录页面
登录后的主页（目前先只显示欢迎语）

## 二期需求(已完成)
### 需求目标
实现笔记管理功能
良好的交互和视觉设计
适配pc端和移动端
暂时只做纯文本笔记，不用富文本
笔记列表显示标题，修改时间，操作按钮，按时间倒序

### 后端
笔记表设计
笔记新增，详情查询，列表查询，修改，删除接口

### 前端
笔记列表页面
笔记新增和编辑入口
笔记编辑页面
删除笔记按钮


## 三期需求(已完成)
### 需求目标
实现待办管理功能
待办列表
待办新增删除修改查询等功能
良好的交互和视觉设计
适配pc和移动端
待办优先级，截止时间，状态等字段

### 后端
代表表设计
待办增删查改接口

### 前端
待办列表
新增待办
修改待办
删除待办

## 四期需求(已完成)
### 需求目标
实现 ai 的基础对话功能
聊天页面
良好的聊天界面视觉设计
历史对话列表

### 后端
llm api 接入
短时记忆对话列表维护
对话记录表设计，用户id，会话id，请求id，对话内容，对话角色，对话时间等
对话记录入库，每条记录为一条对话

### 前端
聊天界面，对话消息显示区域，消息输入区域，发送按钮


## 五期需求(已完成)
### 需求目标
实现基础的对话 agent , 具有总结对话到笔记的能力

### 后端
使用 react 架构，改造后端对话处理逻辑
提供实现对话记录总结 tool , 利用 llm 的 function call 能力，自动总结最近对话并生成笔记

### 前端
无需改动


## 六期需求(已完成)
### 需求目标
在笔记页面实现单次的的任务执行能力，实现在产品各个功能处快速集成ai的基础能力
在笔记详情页面快速生成总结
在笔记页面生成待办

### 后端
实现单次ai任务接口
实现 笔记总结 tool 和 生成待办 tool , 两个tool 内部都不会直接落库或者改什么数据，只会把总结内容或者待办列表返回，这样做主要是为了给前端返回接口稳定的数据
接口传参为 一段提示词和tool名称，将提示词和tool信息注入llm上下文
若ai返回了tool call , 后端就返回状态为成功，将tool 执行后的结果放到 data 字段返回
若ai没有 tool call 而是直接文本返回， 后端就返回状态为失败，将ai的文本回答作为msg字段返回，前端以错误提醒的形式弹出 msg

### 前端
笔记详情页面增加 ‘总结’ 按钮，点击后请求ai单次任务接口， 将 笔记内容+“帮我总结笔记内容” 作为提示词， 和 笔记总结 tool 的 tool name 传给后端，当后端返回成功时，弹出总结内容和 “添加到笔记末尾” 按钮， 当用户点击按钮，就自动将总结追加到笔记尾部
笔记详情页面增加 ‘生成待办’ 按钮， 点击后请求ai单次任务接口， 将 笔记内容+“帮我总结笔记内容” 作为提示词， 和 笔记总结 tool 的 tool name 传给后端， 当后端返回成功时，弹出生成的待办列表 和 一个添加按钮， 默认全选，用户可以取消勾选某些待办，然后点击添加按钮， 将待办保存到待办列表中去。


## 七期需求(已完成)
### 需求目标
实现主会话功能， 进入对话时默认进入主会话，主会话不可删除
实现 @ 笔记功能，在聊天框@后，查询笔记，按时间倒序显示最新5条笔记标题，输入文字过滤相关标题笔记后再显示前5条，点击某一条后，在文本框中高亮显示一个笔记标题，表示引用这篇笔记。服务端解析道消息中有引用某一篇笔记时，把对应的笔记注入到上下文中。

### 后端
实现笔记内容注入，方法是在用户发的消息中，用特殊的标签描述笔记引用，另外实现一个获取笔记内容的tool , tool 的入参为笔记id ,返回笔记内容。所以标签内应该包含笔记名称和笔记id

### 前端
实现@时 笔记列表的查询，构建输入框和聊天记录中的笔记引用高亮渲染，就是文字中显示高亮笔记名，胶囊效果，实际文本是一个包含了笔记名称和id的标签，可以退格删除

## 八期需求(已完成)
### 需求目标
后端的对话上下文目前是限制是最大20条对吧，然后主会话还有个只从最新话题开始加载。 这其中这个20 改成50 吧，然后在加个字数限制，就是如果这50条记录的总字数超过了2万字，就把最前面的若干条从对话记录剔除，然后这里有个重点，不是剔除到2万字以内，而是剔除到2万字除以2，也就是1万字以内，因为如果剔除到刚好2万字一下的话，下一次对话后可能又超了，然后又要剔除。 所以总结来说是设置了上下文大小上线是2万，然后触发一次剔除就剔除到上下文的一半。最后，为了实现这样的需求，你需要在服务中维持一个又状态的会话，而不是每次对话都从数据库查对话记录，然后检查并剔除，内存中的会话要有个过期时间，比如10分钟没有活跃过的会话，就从内存中清除 
就是把内存中的上下文以 jsonl                                                                 
文件的形式往磁盘上存一份，每次对话拉取和更新一下这个文件，这样不用担心会话太多时费内存，也不用处理会话过期的问题。就是不知道会不会和数据库不同步。或者主会话中的新话题如何处理。哦，新话题好 
处理，直接删除这个jsonl文件就好拉，然后对话时没有找到jsonl文件就从数据库重建上下文

## 九期需求(已完成)
### 需求目标
建立知识库管理系统，有新增，删除，查看，编辑知识库的基础功能
新增：类似新增一个文件夹
删除：删除文件夹和文件夹下的内容
查看：查看文件夹内的文档列表，但不需要查看文档内容
编辑：在知识库内上传或者删除文档，不需要编辑文档内容

### 后端
规划一个目录用来维护知识库数据，每个用户有自己的知识库根目录，目录下有每个知识库的目录。应该不需要知识库， 前端需要知识库信息直接从文件系统读取即可

## 前端
实现知识库的增删改查页面。


## 十期需求(已完成)
### 需求目标
知识库索引构建
为 agent 增加知识库查询能力

### 后端
知识库目录下搞一个 config.json 文件， 这个文件不会出现在前端的知识库下的文件列表中，文件中目前可以配置知识库的简介信息
将用户下知识库的名字和知识库的介绍注入到系统提示词中
上传文本文件后，将文本切片，存入向量数据库，存入全文搜索库。 目前先只处理文本文件，比如 txt 和 md , 切片策略先用滑动窗口，窗口大小放到配置文件里
agent 中增加搜索知识库知识的tool , 入参数为知识库名称， 和搜索关键词或者问题。 内部逻辑是搜索向量库和全文搜索，取各自前5条，然后调用重排序 api ,取前5条返回
向量模型用云厂商提供的api
代码示例： 
```bash
~/aiworkspace/tmpwp/gotest/代码说明.md 
```
兼容性要求：目前用嵌入式方案的向量数据库和全文搜索库，以后可能会换用其他中间件，比如独立部署的向量库和全文搜索库，所以要做合理的架构设计

#### api 示例
``` bash
向量模型：
lixubo@MacBook termbot % 
lixubo@MacBook termbot % curl --location 'https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings' \
--header "Authorization: Bearer sk-cb24d621607498sssdfdf" \
--header 'Content-Type: application/json' \
--data '{
    "model": "text-embedding-v4",
    "input": "衣服的质量杠杠的"
}'
返回
{"data":[{"embedding":[0.02258586511015892,-0.08700370043516159, ... ,0.01706676371395588],"index":0,"object":"embedding"}],"object":"list","model":"text-embedding-v4","usage":{"prompt_tokens":6,"total_tokens":6},"id":"b9711baa-df3e-930d-ba37-5932f10639c7"}%       



重排序
lixubo@MacBook termbot % curl --request POST \
  --url https://dashscope.aliyuncs.com/compatible-api/v1/reranks \
  --header "Authorization: Bearer sk-1cb24d6216074sdfgsdfsdf" \
  --header "Content-Type: application/json" \
  --data '{
    "model": "qwen3-rerank",
    "query": "什么是重排序模型",
    "documents": [
      "重排序模型广泛应用于搜索引擎和推荐系统，按相关性对候选文本进行排序",
      "量子计算是计算科学的前沿领域",
      "预训练语言模型的发展为重排序模型带来了新的进展"
    ],
    "top_n": 2
}'

返回
{"object":"list","results":[{"index":0,"relevance_score":0.9079864152388608},{"index":2,"relevance_score":0.7083134275084322}],"model":"qwen3-rerank","id":"13a18c9d-14b5-9f75-b18f-bf4a7167c3da","usage":{"total_tokens":102}}%                                    
```

### 前端
知识库新建和编辑时，可以编辑知识库的介绍信息，比如让用户填写一些介绍知识库中相关知识类型的配置


## 十一期需求(已完成)
### 需求目标
集成网络搜索能力
使用博查 web search api
使用 jina 网页内容获取 api

### 后端
提供 web search tool ,输入搜索信息，输出数组，每个数组元素有url 和 summary 两个字的。 集成 博查 web search api。 接口文档： https://bocha-ai.feishu.cn/wiki/RXEOw02rFiwzGSkd9mUcqoeAnNK 
对话接口增加网络搜索开关参数，启用网络搜索时在 tool list 注入上述 tool

### 前端
增加网络搜索开关，选中时后端启用网络搜索能力

## 十二期需求(已完成)
### 需求目标
实现会话级知识库配置

### 后端
在对话接口中增加知识库信息列表参数，根据这些参数在此轮对话中把知识库列表注入到上下文中，而不是固定注入所有知识库
这里有个问题，目前的对话上下文是会在 jsonl 文件中存档的，那某一轮对话用户改变了知识库列表，会不会和 jsonl 文件中的内容冲突？还是会所 jsonl 中只保存对话信息，不保存系统提示词。每一轮对话都重新构建系统提示词？写代码之前这个要搞清楚

### 前端
联网搜索 按钮同一行增加一个 知识库 按钮，单击一下弹出一个知识库列表，默认都不选，用户可以勾选几个，然后下一次对话时，把这几个知识库和简介列表传到后端

## 十三期需求(已完成)
### 需求目标
集成QQbot 通知

### 后端
#### 示例代码：

基本流程：需要用户qq先给qqbot 发消息， 后端起长连接监听qq用户消息，从中解析到 openid , 消息结构示例：
{"event":"C2C_MESSAGE_CREATE","message_id":"ROBOT1.0_BOoD6lyPaO1l2uQOzixxqurYlfZb1YDTVIc1Odw0PMNLt5Zr2i3vHf0T2yi8uSQYm5TFvRiK-QPuFGFMY2h7CRcbASgFd9.ahn-i8UOMWow!","timestamp":"2026-06-03T21:15:22+08:00","user_openid":"D107BFFBA191BA0905E8A97D279985EC","content":"你好"}

接受qq用户消息，拿到qq 用户openid
~/aiworkspace/tmpwp/pythontest/listen_qqbot_messages.py
给 qq 用户发送消息
~/aiworkspace/tmpwp/pythontest/send_msg.py

#### 接入 qqbot 流程
1. 用户在自己的 qq 开放平台创建机器人，拿到 appid 和 appSceret
2. 到本系统页面录入 qq机器人 appid 和 appSceret
3. 用户在页面点击接入， 后端参考listen_qqbot_messages.py 启动一个长连接，监听qq消息， 然后生成一个随机数返回页面，用户给页面上看到随机数后，给qq机器人发送这个4位随机数，后端收到这个消息，根据随机数映射关系找到用户账户，就能把 qq 的openid 和用户管理起来。

### 前端
页面右上方的用户名可以点击，点击后弹出菜单，菜单里有个绑定qqbot 按钮， 统计把退出也放到这个菜单里
单击qqbot 按钮后，弹出窗口，有个填写 appid 和 appSceret的表单
表单提交后，有个开始绑定的按钮，点击后后端会启动长连接并返回一个4位随机数
然后轮询一个后端接口，后端会在监听到消息成功拿到openid时会在这个接口中返回已经绑定的状态，前端显示绑定成功
如果绑定成功前，前端关闭窗口，就停止轮询

## 十四期需求(已完成)

### 需求目标
实现qq内通过机器人和知玄对话

### 后端
用户配置开启 qq 对话后，后端拿到 用户的 qq  appid 和 appSceret 启动一个 websocket 
用户 qq 内发消息， 被后端监听到， 后端把这个消息发送给agent , agent 处理消息后，发送消息给 qqbot
发 qq 消息的代码可以参考 
```bash
~/aiworkspace/tmpwp/pythontest/send_msg.py
```

#### 架构设计
这里要把 agent 对话主逻辑和外部入口隔离开， 目前有 web 端和 qq 两个外部客户端，用同一套 agent 逻辑
agent 对外暴露统一的对话函数入口
agent 外需要有一层 gateway , 这一层负责对话消息的路由。 
gateway 对外暴露对话函数入口，函数中除了当前对话有的参数外，还需要有个消息来源接口
gateway 收到对话请求后，首先根据对话请求中的 agent id , 把消息放入对应 agent 的待处理消息队列，agent 的待处理消息队列之前没有，需要新实现
然后 gateway 检查 agent 目前是否正在处理消息，如果正在处理，方法就直接结束，如果没有处理，就让 agnet 开始异步处理消息队列中的消息，然后自己的方法结束
agent 被gateway 唤起处理消息后，就开始从消息队列拉起一条消息进行处理，梳理完一条，再自动拉下一条，注意，只有agent处理过的消息，才能进会话jsonl文件和数据消息记录
当 agent 每处理完一条消息，就把消息结果和消息来源告诉gateway , gateway 根据消息来源判断是吧结果返回给web前端，还是发给qq
当 agent 发现消息队列位空时，就停掉当前处理消息线程或者协程。 当下次有消息来时， gateway 发现 agent 停下来了，自然会拉起它

问题：gateway 怎么把消息推给web前端? 可以在前端发消息后，一直维护出这个长连接，当 gateway 拿到agent回复后，可以找到这个 长连接，然后把消息推回去，然后结束连接

#### 关于会话
qq 的消息默认进入主会话，所以qq只会看到从qq内发起和收到的对话记录，而web端主会话因为是从数据库拉取消息记录，所以可以看到所有对话记录，这没关系，符合产品设计逻辑

### 前端
可能要配合后端做长连接的适配，也可能不用做
绑定 qq 弹出内，在已绑定的情况下，显示一个 开启qq对话 的开关，开启时后端才起websocket , 不开时 后端就把 wensocket 关掉。


## 十五期需求(已完成)
### 需求目标
实现各类参数的整理和配置化

### 后端
梳理各类需要配置的参数
数据库配置
llm api , model , key
向量模型 api , model , key
整理用户文件系统，按 用户 -> 业务（知识库，会话文件等等） 整理文件目录。 系统级目录地址要可以配置，系统级目录下是各个用户的私人文件
其他相关需要配置化的参数梳理

## 十六期需求(已完成)
### 需求目标
在使用 qqbot 对话场景中，qq 应用内无法实现联网搜索，知识库开关，@笔记等
要在对话中实现知识库，笔记等内容的注入

### 后端
新增 查询知识库列表 tool 返回知识库名称和简介, 查询笔记列表 tool ，返回笔记标题列表
qq 对话请求过来时，自动注入联网搜索tool, 识库列表 tool, 查询笔记列表 tool ， 知识库内容搜索tool , 查看笔记 tool ，创建待办

### 前端
无需改动

## 十七期需求(已完成)
### 需求目标
实现定时任务和主动提醒

### 后端
增加定时任务表，核心字段有 任务名称，任务类型，执行频率 cron , 任务参数， 是否开启qq通知等
增加任务执行日志表，核心字段有 任务id , 任务执行结果等
增加任务执行器接口，定义执行任务的抽象方法
目前实现一种任务类型为 agent 任务，参数是自然语言文本，任务逻辑是默认用户把参数提交给主会话，agent 处理这条消息，处理完把agent返回文本作为接口存入任务执行日志
增加 qq 通知 tool , 在用户开启 qq 通知的状态下注入上下文， tool 如参是用户 id 和文本消息内容, tool 的描述要说明只在必要且用户明确要求 qq 通知时才使用，防止打扰用户
在上下文系统提示词中注入 用户id, 以便于在llm调用 qq 通知 tool 时使用

### 前端
页面新增定时任务选项卡
定时任务新增、编辑、删除、列表查询等交互逻辑
定时任务类型通过查询后端接口获得类型枚举，页面交互为下拉选择

## 第十八期需求
### 需求目标
集成微信通信，先集成普通文本通信

### 后端
参考代码
``` bash
~/work/openitem/openclaw-weixin-main/weixin_text_echo.go
~/work/openitem/openclaw-weixin-main/weixin_text_echo.py
~/work/openitem/openclaw-weixin-main/WEIXIN_ILINK_PROTOCOL.zh_CN.md（文档较大，谨慎加载）
```
绑定流程：
生成二维码 ，微信用户扫码，服务端拿到微信数据，绑定用户
gateway 层增加微信消息路由，类似qq, 也是发到主会话
通知提供 发送消息到微信的 tool
注意用户微信发消息过来的时候，好像是要把最新token保存下来，有token 才能给微信回消息

### 前端
绑定qq旁边增加绑定微信按钮
点击绑定微信按钮后，弹出二维码，后端会把二维码中的数据和当前用户临时关联
前端轮询绑定状态
用户扫码后，后端拿到用户微信信息,把微信信息和用户绑定，绑定成功
前端发现绑定成功后，关闭二维码，显示开始微信对话的开关


## 第十九期需求(已完成)
### 需求目标
模型可能下架或因为额度等问题短期内不可用，要支持文本模型自动切换可用模型
模型默认优先级：
gui-plus-2026-02-26
qwen3.6-plus
qwen3.6-plus-2026-04-02
qwen3.6-flash
qwen3.6-35b-a3b
qwen3.6-flash-2026-04-16
qwen3.6-max-preview
kimi-k2.6
qwen3.6-27b
qwen3.5-plus-2026-04-20
deepseek-v4-pro
deepseek-v4-flash
qwen3.7-max-2026-05-20
qwen3.7-max
qwen3.7-max-preview
qwen3.7-max-2026-05-17
qwen3.7-plus-2026-05-26
qwen3.7-plus

### 后端
使用llm api 时按需求中的模型列表顺序进行调用，当调用api报错时，自动切换到下个api重试，同时把当前调用失败的模型放到列表的最后一位，调整顺序后的模型列表要持久化，以防止之后多次重试
最好想写个测试代码，看看模型不可用时返回的结果是什么样的，目前提供一个不可用模型进行测试：deepseek-v3.2-exp
默认模型列表要可以在配置文件中配置
问题：没想好重试后调整模型顺序后把最新的顺序持久化到哪里，持久化的话，之后改了配置文件怎么刷新，还是不要持久化？

## 第二十期需求(已完成)
知识库集成图片识别和识别能力
web 页面支持ai回复图片

### 后端
知识库中上传图片时，调用视觉理解 api 将图片内容转文本，然后把api返回的整段文本直接作为一个 chunk 做存储，向量化，和全文索引
agent 查询知识库时，召回并返回给 llm 的chunk 片段中要标明来源文件类型和文件相对于用户知识库目录的相对路径，比如图片的位置在 /knowledge_bases/1/工具书/apple.png, 那么给llm的信息中就标明类型是 img , 路径是  knowledge@工具书/apple.png 。 其中 @ 前面的 knowledge 表示数据在知识库中， 因为以后可能还会总其他地方获取图片。后面的 工具书/apple.png 表示当前用户的知识库根目录下的相对文件路径
然后在 search_knowledge_base tool 的返回数据逻辑中加点东西，如果发现命中了图片数据，就在返回的文本中加一写提示，比如：
“找到如下结果，如果需要返回图片，请返回 [image:图片名称:文件路径]，示例 [image:apple:knowledge@工具书/apple.png]\n (之后是相关内容)”
具体的提示词格式可以在斟酌，主要目的是让llm 在合适的时候可以按固定的格式把图片信息返回前端，然后前端就可以从文本中把图片信息提取出来，然后请求后端拿到并展示图片
后端提供一个接口，接受类似 knowledge@工具书/apple.png 的参数，knowledge 表示从知识库取数据， 工具书/apple.png 表示从当前用户的知识库下按此路径取数据
补充：
调用图片理解 api 时先对图片进行压缩，确保图片文件大小小于1m
调用图片理解 api 模型的提示词中限制不得超过500字，示例提示词：描述一下图片内容,返回json 数组，两个字段, text:文字内容(可为空)，content:图片描述(不超过500字)"
注意这里让 api 返回json ,但做向量索引和全文索引时都不用解析json，直接存即可，让 api 输出json 只是方便以后用
api 返回的图片的理解内容都用文件保存下来以便于以后再查和用，文件名是图片文件名+ .ocr 。类似 apple.png.ocr, .ocr 文件不要返回给前端
``` bash 
# 调用api 识别图片的代码示例：
# base_url 和 api_key 和文本模型是一样的，但配置要独立出来，方便以后切换
~/aiworkspace/tmpwp/pythontest/vl.py 
```
可用的视觉理解模型：
gui-plus-2026-02-26
qwen3.6-plus
qwen3.6-plus-2026-04-02
qwen3.6-flash
qwen3.6-35b-a3b
qwen3.6-flash-2026-04-16
kimi-k2.6
qwen3.6-27b
qwen3.5-plus-2026-04-20
qwen3.7-plus-2026-05-26
qwen3.7-plus
按这个顺序配置到配置文件，调用时如果失败自动切换下一个模型

### 前端
知识库本就支持上传任意类型文件，不用改
聊天界面收到ai回复后，解析其中的图片引用，然后调用一个接口，把类似 knowledge@工具书/apple.png 的内容传给后端，获取并展示图片

## 第二十一需求(已完成)
### 需求目标
支持立即停止 agent 工作
支持斜杠命令，主要用于 qq ,微信 等没有复杂前端交互的通道，web 页面也可以用
## 后端
agent 支持停止信号, 终止 agent 循环
解析用户消息若为 / 开头，则执行对应命令，目前支持 /stop , 业务逻辑为向 agent 发送停止信号，agent 每一轮循环前先判断是否有停止信号，有则停止

### 前端
由于发送消息在 ai 回复过程中无法发送消息，故无法发送 /stop , 可以在ai回复过程中把 发送按钮改为 “停止” 按钮，点击后自动向后端发送 /stop 命令


## 第二十二期需求(已完成)
### 需求目标
web 聊天页面支持发送图片

### 后端
文件系统中开辟一个目录存储用户在对话中上传的各类文件
当用户在对话中上传图片时，把图片上传到这个目录中
对于 web 页面中的交互，是调用接口先上传图片，然后后端返回图片的索引标识，比如 upload@session_1/example.png , 这时只上传了图片，发送按钮依然置灰，然后用户在消息框输入文本，比如“图片里有什么” ， 点击发送时， 前端实际拼接消息为 "[image:example:upload@session_1/example.png]图片里有什么" , 然后前端历史消息显示区域会展示这套消息，并自动 调用接口查询 upload@session_1/example.png 图片数据和展示图片数据。 类似之前的知识库的图片展示。
后端新增一个理解图片的 tool  , tool 的入参有两个，一个是图片标识， 类似 upload@session_1/example.png 这种，另一个参数是 提示词，用于放置用户问题，当 提示词 参数为空时， tool 内部自动设置默认提示词为 “描述一下图片内容,返回json 数组，两个字段, text:文字内容(可为空)，content:图片描述(不超过500字)” ， 然后 tool 内部调用视觉理解模型理解图片，把视觉理解模型的返回座位 tool 的返回直接返回。 注意，这里我写的默认提示词让模型返回json ,但tool 内部无需解析json，直接返回即可。

### 前端
消息输入框上方知识库右边加一个按钮 “+” ， 点击后显示菜单，里面目前只有一个 "图片"，点击点击图片后让用户选择本地图片， 选择后调用接口上传图片，得到图片标识
图片预览：
1. 布局结构
采用垂直堆叠的块级容器。该组件位于文本输入区域的正上方，作为独立的视觉区块占据空间，将下方的文字输入区向下挤压，形成“上图下文”的分层结构。
2. 视觉样式
呈现为圆角矩形卡片。具有独立的边框描边与背景色，内部通过固定比例的裁剪方式渲染图片缩略图，确保视觉饱满且整齐。
3. 交互状态
处于预发送暂存态。卡片右上角悬浮显性的移除图标，提供即时的撤销操作入口，允许用户在最终提交前对已选中的媒体资源进行删除或替换。


## 第二十三期需求(已完成)
### 需求目标
稳定的文件 读，写，改 是 agent 的基本能力， 集成这些等tool, 先实现能力，
文件列表的探索能力
### 后端
找了一些其他项目中的实例代码可以参考

``` bash
claude-code 实现：
Read: ~/work/openitem/claude-code/src/tools/FileReadTool/FileReadTool.ts
Write: ~/work/openitem/claude-code/src/tools/FileWriteTool/FileWriteTool.ts
Edit: ~/work/openitem/claude-code/src/tools/FileEditTool/FileEditTool.ts
NotebookEdit: ~/work/openitem/claude-code/src/tools/NotebookEditTool/NotebookEditTool.ts

openclaw 实现：
 ~/work/openitem/openclaw-main/src/agents/sessions/tools/read.ts
 ~/work/openitem/openclaw-main/src/agents/sessions/tools/write.ts
~/work/openitem/openclaw-main/src/agents/sessions/tools/edit.ts
~/work/openitem/openclaw-main/src/agents/sessions/tools/path-utils.ts
~/work/openitem/openclaw-main/src/agents/sessions/tools/file-mutation-queue.ts
~/work/openitem/openclaw-main/src/agents/sessions/tools/edit-diff.ts
~/work/openitem/openclaw-main/packages/agent-core/src/harness/utils/truncate.ts
```

获取目录下文件列表的tool,的参考代码
```bash
claude-code 实现：
~/work/openitem/claude-code/src/tools/GlobTool/GlobTool.ts
~/work/openitem/claude-code/src/utils/glob.ts
~/work/openitem/claude-code/src/tools/BashTool/BashTool.tsx
~/work/openitem/claude-code/src/tools/FileReadTool/prompt.ts

openclaw 实现：
~/work/openitem/openclaw-main/src/agents/sessions/tools/ls.ts
~/work/openitem/openclaw-main/src/agents/sessions/tools/find.ts
~/work/openitem/openclaw-main/src/agents/sessions/tools/grep.ts
```

## 第二十四期需求(已完成)
### 需求目标
实现 qq 端的接收和发送图片功能

### 后端
参考目前 web 端页面聊天的实现,保存和索引图片
gateway 中对qq通道的消息做处理
当收到qq图片消息时，先存本地，然后构造图片索引标签发给 agent
当 agent 回复图片时，agent 回复的是图片标签， gateway 这一层要做处理： 比如agent 回复 “文字1<图片>文字2” ，gateway 这一层先发送 文字1 ， 然后解析和发送图片，然后发送文字2

```bash
# qqbot 发送和接收图片的代码参考：
~/work/openitem/openclaw-qqbot-main/QQ_IMAGE_API.md
```


## 第二十五期需求(已完成)
### 需求目标
实现计划管理功能
计划有标题和详情，计划下可以关联笔记和待办
笔记和待办模型增加 plan_id 字段关联计划
在外部笔记和待办列表中，关联了计划的条目仍可正常看到和操作，功能不增不减

### 后端
用 Plan 命名，避免和定时任务 Schedule 冲突
计划表设计：标题、详情、状态、用户id、创建时间、更新时间
笔记表和待办表增加可空的 plan_id 字段关联计划
计划新增、详情查询、列表查询、修改、删除接口
笔记和待办列表查询支持按 plan_id 过滤
删除计划时只解除关联（将关联笔记和待办的 plan_id 置空），不删除笔记和待办

### 前端
页面增加"计划"标签
计划列表页面：展示计划列表，新建计划入口，删除计划
新建/编辑计划页面：上方标题输入框，下方详情文本输入区，保存落库
计划详情页面：页面内分两个区块平铺展示该计划下的笔记和待办，可在计划内新建笔记和待办，新建时自动关联该计划


## 第二十六期需求(已完成)
### 需求目标
核心理念："一切软件都是数据库的华丽 GUI"。如果 agent 能直接操作数据库，理论上它就能承载大部分 CRUD 类软件的功能，而不必为每个新需求都单独开发 tool。
给 agent 提供一个独立于系统主库的 SQLite 数据库，作为它的"自留地"，agent 可以在里面自由地建表、查询、增删改数据，承载业务里没预想到的结构化数据需求（例如读书清单、项目进度追踪、自定义数据管理等）。
通过物理隔离的方式，避免 agent 误操作影响系统核心数据（用户、会话、笔记、待办等表）和其他用户的数据。

### 设计决策
几个关键决策点先想清楚，避免实现时反复：
1. agent 库独立用 SQLite，不跟随系统主库类型。即使主库是 MySQL，agent 库也是 SQLite。理由：SQLite 一个文件就是一个库，配置和备份都简单；DDL/DML 不用考虑方言；agent 数据量级一般不大，SQLite 完全够用；SQLite 本身就是结构化的文件存储，符合"基于文件存储"的思路，同时具备查询能力（纯 JSON 文件只能存不能高效查，撑不起"承载大部分软件功能"的目标）
2. 每个用户一个独立的 SQLite 文件（如 agent_<userid>.db），物理隔离用户数据。理由：不靠"工具层自动拼 WHERE user_id" 来做隔离——SQL 字符串拼接处理 JOIN、子查询、UNION、CTE、别名等场景不可靠；SQL 解析器方案虽然能做但引入大依赖且复杂度高；每用户一个文件物理隔离最干净，agent 写啥 SQL 都对
3. agent 库与系统主库完全隔离。agent 看不到也改不了 users/sessions/chats/notes/todos 等系统表。agent 需要查询系统数据时仍走现有工具（list_todos、list_notes 等）
4. agent 在自己的库内可以执行任意 DDL 和 DML（含 DROP）。因为砸了也只影响 agent 自己的数据，不影响系统稳定性和他人数据，风险可控

### 后端
agent 数据库管理：
- 数据库文件存放在 <DataDir>/agent_db/ 目录下，每个用户一个 agent_<userid>.db 文件
- 维护一个用户到 *gorm.DB 的映射（sync.Map），用户首次操作时打开连接并缓存，避免每次请求重复打开
- 使用 SQLite 驱动，DSN 开启 WAL 模式、busy_timeout、foreign_keys
- 不做 AutoMigrate，表结构完全由 agent 自行管理

新增 3 个 agent tool（暴露给 agent 使用）：
1. dump_schema：一次性返回所有表的建表语句（SELECT name, sql FROM sqlite_master）。初期表少时一个调用即可看清全局，替代 list_tables + describe_table 的组合，减少 tool call 轮次和上下文 token 消耗
2. query：执行只读查询（SELECT/WITH 开头）。安全约束：禁止分号（防多语句）；末尾无 LIMIT 时自动追加 LIMIT 200；单字段值超 500 字符截断；5s 超时保护
3. execute：执行 DDL/DML（CREATE/INSERT/UPDATE/DELETE/DROP/ALTER 等）。安全约束：禁止 ATTACH/DETACH（防绕过物理隔离）、禁止 PRAGMA（防篡改连接行为）；5s 超时保护；返回影响行数

工具执行约束：
- query 严格只读，execute 可写可改可建表可删表
- user_id 由服务端注入（从认证会话获取），不经过 LLM 参数，agent 无法伪造或越权
- list_tables 和 describe_table 的代码保留但不暴露给 agent，后期表多了如需拆分查看时可重新启用

系统提示词调整：
- 告知 agent 拥有独立的 SQLite 数据库，可自由建表存储自定义结构化数据
- 首次使用前先 dump_schema 了解现状，避免重复建表
- 说明该库与系统笔记/待办/计划数据是分开的，查系统数据用对应工具
- 鼓励 agent 主动记录用户画像信息（爱好、职业、年龄、心情、重要事件等），即使用户没有明确要求
- 现有表中没有合适的表时，agent 可用 execute 自行新建表

### 前端
无需改动


## 第二十七期需求：动态上下文注入优化(已完成)
### 需求目标
当前系统提示词中包含 `当前时间`，该值每次请求都不同，导致整个系统提示词无法命中 LLM 提供商的 prompt cache，长对话时白白多算大量 token。同时时间信息放在系统提示词最前面，对话越长 LLM 对它的注意力越弱（recency bias），agent 回答时间相关问题时可能用的是过时信息。

### 设计方案
将 `当前时间` 从系统提示词中摘出，改为在发往 LLM 之前**伪造一组 tool call（assistant 发起 + tool 返回结果）**，插入到 messages 数组末尾。这样动态信息以 tool result 的身份出现在对话最末尾，不污染用户消息原文，角色交替也合法（user → assistant(tool_call) → tool(result) → LLM 生成回复）。用户ID 和知识库列表在各自作用域内（用户/会话）是稳定的，保留在系统提示词中不影响缓存。

### 后端
- buildSystemPrompt 不再拼接 `当前时间`，只保留用户ID 和知识库列表等相对稳定的信息
- processMessage 中组装 messages 数组后，在末尾追加两条伪造消息：
  - assistant 消息：携带一个 tool_call（如 `get_context`，无需在 tools 列表中注册）
  - tool 消息：返回当前时间等动态信息（JSON 格式）
- 伪造的 tool call 不落库：每次请求临时组装，用完即弃，不写入 JSONL 上下文缓存
- 未来可扩展：往伪造的 tool result 中追加更多动态信息（如用户在线状态、会话消息数等）

### 收益
- 系统提示词变为静态（按用户/会话维度），可被 prompt cache 命中，节省长对话 token 成本
- 动态时间信息以 tool result 身份位于消息末尾，LLM 注意力更强，时间感知更准确
- 不污染用户消息原文，语义清晰
- ReAct 循环中每次 LLM 调用都携带最新时间，而非 processMessage 入口时的快照

### 前端
无需改动


## 第二十八期需求：对话历史文件化持久化(已完成)
### 需求目标
当前对话历史采用 DB（Chat 表）+ JSONL 双写：JSONL 存全量（含 tool call），DB 只存 user/assistant 文本。DB 的 Chat 表本质上是 JSONL 的降级残缺副本——缺少 tool call 信息，且需要维护 schema 一致性。聊天记录是追加-only、按时间排序、按会话隔离的日志型数据，天然适合文件存储而非关系型 DB。

本次改造放弃 Chat 表，改为用独立的持久化 JSONL 文件存储完整对话历史（含 tool call），按天分片。现有的 context_cache JSONL 保持不变，继续作为内存上下文快照使用。

### 存储结构
两套 JSONL 职责分离，类比 MySQL 的 binlog 与 redo log / buffer pool 的关系：

| | context_cache（现有，保留） | chat_history（新增） |
|--|--|--|
| 类比 | redo log / buffer pool | binlog |
| 路径 | `<DataDir>/context_cache/{sessionID}.jsonl` | `<DataDir>/chat_history/{sessionID}/{date}.jsonl` |
| 定位 | 内存上下文快照，服务当前对话 | 完整持久化记录，数据真相源 |
| 内容 | 裁剪后的上下文（旧消息会被从头砍掉） | 全量（对话 + tool call + tool result） |
| 写入方式 | 读全量→追加→裁剪→重写（现有逻辑） | 纯追加写（`O_APPEND`，一行一条） |
| 生命周期 | 可丢失、可从 chat_history 重建 | 永久保留 |
| 可靠性要求 | 不要求，丢了无所谓 | 要求高，这是唯一持久化层 |

两者并行写、各管各的：每条消息同时追加到 context_cache（经裁剪管理）和 chat_history（纯追加）。context_cache 丢了可以从 chat_history 恢复，chat_history 丢了才是真丢数据。

日期格式：`2026-06-14.jsonl`（按自然日分片，以消息产生的服务器时间为准）。

### 写入
- 每条消息（user / assistant / tool_call / tool_result）产生后，立即追加写入当天对应的 chat_history JSONL 文件
- 文件不存在时自动创建（含父目录）
- 写入方式：`os.OpenFile(path, O_APPEND|O_CREATE|O_WRONLY, 0644)`，每条消息 `json.Marshal` 后追加一行 + `\n`
- 不读取旧内容、不裁剪、不重写——纯追加，写入性能和可靠性最优
- 不需要加锁：Agent 的 processLoop 串行处理同一用户的队列，同一 session 不存在并发写入同一文件的场景
- 定时任务触发的消息也正常写入 chat_history，不做任何特殊标记或区分

### 加载场景一：Web 页面展示
- 用户打开会话时，展示最近 20 条对话（1 条对话 = 1 个 user 消息 + 1 个 assistant 回复，为一轮）
- tool call / tool result 不计入 20 条计数，也不展示
- 加载逻辑：按日期倒序遍历 chat_history 文件，逐行正向读取（`bufio.Scanner`），内存中维护滑动窗口只保留最近 20 轮对话，累计够后停止。不全量加载文件到内存
- 翻页加载更早的历史时，继续向前读取更多日期文件

### 加载场景二：Agent 上下文构建
- 每条消息处理时，从 context_cache 读取上下文（现有逻辑不变，快速路径）
- 如果 context_cache 不存在或损坏，从 chat_history 重建：按日期倒序遍历，逐行读取累计最近 20 轮对话（含其中的全部 tool call / tool result），写入 context_cache
- 无论从哪条路径加载，读到的上下文直接使用，不剪裁；上下文体积控制改由第三十四期的 LLM 压缩在写路径处理（超过阈值时压缩历史、保留近 5 轮原文）
- 孤儿 user 消息（assistant 回复失败的半轮对话）正常加载，不做特殊处理

### Chat 表处理
- 停止写入：processMessage 中不再 `database.DB.Create(&userMsg)` 和 `database.DB.Create(&assistantMsg)`
- 停止读取：GetOrLoad 的 DB fallback 逻辑移除，改为从 chat_history 加载
- 表保留不删，AutoMigrate 保留（不影响）
- 存量数据不迁移，旧对话留在 DB 里不管，新对话走 chat_history

### 收益
- 单一持久化数据源，不再有 DB 和 JSONL 不一致问题
- 天然存全量（tool call、result 都在），无需给 Chat 表加字段
- 纯追加写性能好、可靠性高（不怕进程崩溃丢数据）
- 零 schema 迁移成本——llm.Message 加字段，JSONL 自动带上
- 按天分片，单文件不会无限膨胀，便于管理和备份

### 前端
- 会话历史加载接口从读 DB 改为读 chat_history JSONL
- 其余无需改动


## 第二十九期需求(已完成)
### 需求目标
web 端 ui 优化
- 采用三段式侧边导航 + 二级分类侧边栏 + 主内容区 的 工具复合布局
- 最左侧：一级全局侧边导航（固定窄栏）。目前的标签页形式换成左侧纵向排列的一列图标，顺序：对话、待办、笔记、计划、知识库、定时任务
- 中间栏：二级分类侧边栏（中等宽度子导航）。点击某个图标后，图表列右侧弹出对应的列表，比如对话就是会话列表，待办就是待办列表
- 最右侧：主内容展示区（自适应宽主面板）。点击列表中的某一项，右侧剩余大片区域显示详情， 比如会话聊天界面、笔记详情，待办详情等

## 第三十期需求：旁路记忆系统(已完成)
### 需求目标
在主 agent 对话过程中，异步启动一个记忆 agent，自主判断是否有值得长期记录的内容，有则通过工具调用写入记忆。记忆可在后续对话中被召回并注入上下文，实现个性化。

### 设计要点
- 异步执行：记忆 agent 不阻塞用户响应
- 复用主 agent 的对话循环，记忆 agent 只替换自身的提示词、工具集与工具执行逻辑
- 存储采用 数据库 + 向量索引 混合方案：
  - 结构化表存记忆字段（类型、内容、标签、来源会话、时间）
  - 同步写入向量索引用于语义召回
- 记忆 agent 工具集：保存、更新（合并去重）、搜索（查重）
- 上下文投喂：传递完整对话历史（仅 user/assistant 文本），剔除工具调用类消息，保持前缀稳定以命中 LLM 前缀缓存
- 记忆 agent 串行调度：独立的调度器按用户维度维护队列。每用户容量上限 2（1 运行 + 1 等待），新任务到达时若等待槽已占则替换旧任务。保证同一用户记忆整理串行执行，避免并行写入重复记忆
- 批处理节流（不再每轮都写）：按会话记录一个"已处理到的消息时间戳"作为 checkpoint，累计达阈值的用户消息条数后才拉起一次记忆 agent（阈值默认 3、可配置）；批处理时把自 checkpoint 以来整段新对话（已剔除工具调用）投喂给它，提交即推进 checkpoint（best-effort，失败不回退）。checkpoint 与召回窗口同属一个会话状态文件，避免两者互相覆盖（见第三十二期）
- 首次/部署迁移保护：当 checkpoint 仍为初始值时，只处理当前这一轮消息，避免一次性回放全部历史
- 记忆 agent 提示词要点：写入粒度适中（一个事实一条记忆，避免过细导致重复）；同轮刚保存的记忆不再用 update 反复改；查重时鼓励在一次工具调用轮次里并行发起多个搜索，减少请求轮次

### 后端
- 记忆表设计
- 记忆 agent 实现（专用提示词、工具集、工具执行）
- 主 agent 完成回复后按批处理规则（checkpoint + 用户消息条数阈值）异步触发记忆 agent

### 前端
- 记忆列表页：查看、删除记忆


## 第三十一期需求(已完成)
### 需求目标
让用户在页面上维护一组可开关的提示词（skill），稳定地触发某些行为（例如"用户说今天的运动量时，记录到 sport_records 表"）。skill 摘要随每轮对话注入上下文，详情经 load_skill tool 懒加载，避免常驻占用上下文。skill 全局生效，不限数量，不预置内置。

### 设计要点
- skill 列表信息不进 system prompt，而是追加到第二十七期已有的伪造 tool call（get_context）的 tool result 中，避免污染稳定前缀、保住 prompt cache
- load_skill tool 放进 tool list，其 description 承载 skill 机制说明与调用时机；伪造 tool call 的 result 里只放纯数据（name/summary/has_detail）
- 原则：tool 的 description 既说明功能也说明调用时机，system prompt 永远不描述 tool 用法（tool 会越来越多，不应挤在 system prompt 里）
- 同时放开此前被过滤的 agentDB 三 tool（dump_schema/query/execute），并清理死代码 agentDBPrompt（该常量从未被引用）

### 后端
- skills 表：id、user_id、name、summary、detail(可空)、enabled、sort、created_at、updated_at；AutoMigrate 列表加入 model.Skill
- skill 增删改查接口（列表、新建、修改、删除、开关）
- buildSystemPrompt 不改动（skill 不入 system prompt）
- processMessage 中伪造 tool call 的 tool result 由 {"current_time":"..."} 扩展为 {"current_time":"...","skills":[{name,summary,has_detail}]}，仅取 enabled=true，按 sort 排序
- 新增 load_skill tool：入参 name，出参该 skill 的 detail；detail 为空时返回明确提示（列表的 has_detail 字段已避免绝大多数无效调用，此为兜底）
- tools 数组追加：dumpSchemaTool、queryTool、executeSQLTool、loadSkillTool
- 删除 server/gateway/tools.go 中未引用的 agentDBPrompt 常量

### 前端
- 对话页会话列表 header（ChatView.vue:7 当前的 100% 宽「新建对话」按钮）改为两个按钮并排：「新建对话」「Skill 管理」，充分利用该行空间
- 点击「Skill 管理」弹出 skill 管理弹窗：列表展示 name、summary、是否启用（开关）、有无详情（标记）、编辑、删除；支持新增
- skill 编辑表单：name、summary、detail（多行文本）、enabled、sort


## 第三十二期需求：对话记忆召回与注入(已完成)
### 需求目标
补上主对话 agent 的记忆读取链路。每轮对话用用户消息原文向量召回记忆，合并进会话级 LRU 窗口（最大 10 条），窗口内容随伪造的上下文注入 tool call 进入上下文，让重要记忆在整个会话持续可见、不被当前 query 漂走。同时主 agent 增加 search_memory tool，供 LLM 主动检索。

### 设计要点
- 现状：记忆系统此前只写不读，记忆搜索的唯一调用方是记忆 agent 的查重 tool，主 agent 既不在 system prompt 也不在伪 tool call 注入记忆。本期补上"读"链路
- 两条独立链路，互不影响：
  - 被动召回（每轮自动）：用户消息原文 → 向量召回 top 5 → 按相关度阈值过滤 → 合并进 LRU 窗口 → 窗口快照作为记忆列表注入伪 tool call
  - 主动搜索（LLM 调用）：search_memory 作为正常 tool call，结果直接返回给 LLM，不进窗口、不走任何合并逻辑
- LRU 窗口（大小 10）：命中的提升到队首、新记忆插入队首、超过 10 条踢队尾；按记忆 ID 去重
- 空结果占位淘汰：某轮召回过滤后无任何命中时，"空结果"也算一种结果，在窗口里占一个位置（用一个占位符表示），占位符不去重、逐次累计，随空轮累积把窗口尾部的旧记忆逐步挤出。目的是用户持续聊新话题时旧记忆能被慢慢"忘掉"，避免空轮让老记忆永远赖在窗口里。占位符只用来驱动淘汰，注入上下文时过滤掉、不展示
- 会话级运行时状态（召回窗口 + 记忆 agent 的处理 checkpoint）合并到同一个按会话 ID 命名的状态文件，避免窗口与 checkpoint 各写各的互相覆盖；不碰对话历史与会话上下文缓存；文件丢了无所谓，下一轮召回自然重建（checkpoint 退化为初始值，下批从头取）
- 相关度阈值：向量召回返回的余弦分数低于阈值的不进入窗口，默认 0.5、可配置；阈值在 topK 之后于应用层执行（底层向量库只按 topK 裁剪，不按分数砍）；过滤前的全部命中及分数打印日志，便于观测分布、调整阈值
- 注入体积：窗口上限 10 条，单条内容超 100 字截断兜底
- 不做上下文剪裁时的全量刷新，靠 LRU 自然老化

### 后端
- 新增会话级记忆窗口的 LRU 管理与文件读写（按会话 ID 持久化）；窗口支持空结果占位（空轮写入占位符、注入时过滤）
- config 增加窗口文件目录与召回阈值两个配置项
- 主 agent 处理消息时追加：用用户消息原文向量召回 top 5 → 按阈值过滤 → 合并进窗口并持久化 → 伪造上下文注入时把窗口快照作为记忆列表注入（每条内容截断 100 字）
- 向量召回需保留分数，新增带分数的搜索方法，不动现有查重用的搜索（记忆 agent 查重行为不变）
- 主 agent 注册 search_memory tool，复用记忆 agent 已有的搜索逻辑，但不碰窗口
- 开新话题时：清理上下文缓存，同时清空召回窗口（旧话题的常驻记忆不再占位），但保留记忆 agent 的 checkpoint，让话题切换前未达阈值的尾巴能在下一批被自然处理
- 数据库无需新增表（窗口是文件不是表）

### 前端
- 无需改动


## 第三十三期需求：知识库文件预览(已完成)
### 需求目标
在知识库的文件列表中，对常见格式提供在线预览，无需下载即可查看内容。第一批覆盖浏览器可原生渲染、前端零依赖的格式；需第三方库解析的 Office 类格式留待后续。

### 设计要点
- 现状：上传的原文文件按"用户/库/文件名"留盘保留，但列表只有"删除"，没有取文件内容的接口。本期补上"读"链路：后端取文件接口 + 前端按类型渲染
- 第一批按"浏览器原生能力"分批，零新依赖：
  - 图片（png/jpg/jpeg/gif/webp）→ 图片预览
  - PDF → iframe 内嵌，用浏览器自带 PDF 渲染
  - 纯文本类（txt/md/log/csv/json 等）→ 拉取文本展示；Markdown 用已有 marked 渲染，其余等宽展示；超大文件只预览前若干行或给出"过大"提示
  - HTML → iframe 渲染
- 安全：用户上传的 HTML 可能内嵌 `<script>`，直接渲染在同源下会引发 XSS（可窃取 cookie、带凭证调 API）。用 iframe 的 `sandbox` 属性：只开放 `allow-scripts`、不开 `allow-same-origin`，脚本可执行（图表类 HTML 能正常展示）但 iframe 为隔离的独立源，访问不到父页面的会话/cookie/接口
- 后端取文件接口按当前用户隔离目录，复用现有的库名/文件名校验防目录穿越；以 inline 方式返回原始字节（预览而非下载），Content-Type 按扩展名识别
- 预览入口：文件列表每项增加"预览"操作，点开弹窗按类型分流渲染；暂不支持预览的格式（如 Office）给出明确提示，便于后续扩展

### 后端
- 新增取知识库文件内容的接口（按 库名 + 文件名 定位），inline 返回原始字节，Content-Type 由扩展名推断
- 复用现有名字校验防目录穿越，按当前登录用户隔离到各自目录
- 文本类预览由前端直接拉取该接口内容，无需单独接口

### 前端
- 文件列表每项增加预览入口
- 预览弹窗按文件类型分流渲染：图片 / PDF iframe / 文本与 Markdown / HTML（sandboxed iframe，仅开 allow-scripts、不开同源）
- Markdown 用项目已有的 marked 渲染，纯文本等宽展示，超大文件限流展示
- 暂不支持的格式给出提示


## 第三十四期需求：会话上下文压缩(已完成)
### 需求目标
把上下文快照原有的按字符数硬截断换成 LLM 压缩：上下文体积超过阈值时，把较早的对话压成摘要、保留最近若干轮原文，在长会话里控制上下文体积、尽量少丢有效信息。压缩只发生在写回快照这一步，读路径与模型调用不引入额外开销。

### 设计要点
- 读写分离：
  - 读路径（ChatWithTools 之前）：从快照读上下文，**读到直接用，不剪裁不压缩**；读不到从历史记录加载最近 20 轮（已有逻辑）
  - 写路径（ChatWithTools 之后）：用本轮最后一次 LLM 调用的 **total_tokens** 判断，未超阈值追加写，超阈值先压缩再覆盖写
- 用 total_tokens 而非 prompt_tokens：模型输出也占上下文且会进下一轮输入；ReAct 循环里上下文只增不减，末次调用的 total_tokens 即本轮峰值，取它判断即可
- 压缩规则：最后 5 条 `role:user` 消息及其间的全部消息（assistant / tool_call / tool_result）**原样保留、不许动**；之前的所有内容喂 LLM 压成 **≤5000 字** 摘要，作为一条 `role:user` 置于快照最前；新结构 = `[摘要] + [原文尾部]`
- 阈值默认 **50000 total_tokens**，可配置
- 边界：超阈值但 user 消息总数 ≤5 时跳过压缩直接追加写（无可压缩内容，常见于单条消息/单次 tool_result 巨大）
- 压缩是单次 LLM 请求（无 tools 无循环）；压缩失败退回追加写，打日志
- 旁路记忆、召回窗口走各自的会话状态文件，与快照互相独立，不受影响

### 后端
- ChatWithTools 追加返回最后一次 LLM 调用的 total_tokens
- 快照模块：读路径去掉字符截断；写路径改为按阈值追加或压缩覆盖；删除旧的按字符截断逻辑
- 新增压缩函数：按"最后 5 条 user 及其间全部消息"切分，前段摘要、后段原样
- config 增加压缩阈值（默认 50000）与摘要字数上限（默认 5000）

### 前端
- 无需改动























