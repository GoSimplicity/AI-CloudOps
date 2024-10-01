import os

from dotenv import load_dotenv
from openai import OpenAI

load_dotenv()
api_key = os.getenv("OPENAI_API_KEY")

client = OpenAI(
    api_key=api_key
)

completion = client.chat.completions.create(
    model="gpt-4o-mini",
    response_format={
        "type": "json_object",
    },
    messages=[
        {
            "role": "system",
            "content": '你现在是一个JSON对象提取专家，请参考我的JSON定义输出JSON对象'
                       '示例: {"service_name":"","action":""}，'
                       'action可以是:get_log, restart, delete'},
        {
            "role": "user",
            "content": "请重启我的payment服务"
        }
    ]
)

print(completion.choices[0].message.content)
