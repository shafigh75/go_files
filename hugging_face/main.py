# sample use case of a chat completion using requests and no additional modules using gemma2

import requests

headers = {
    'Authorization': 'Bearer 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',
    'Content-Type': 'application/json',
}

json_data = {
    'model': 'google/gemma-2-2b-it',
    'messages': [
        {
            'role': 'user',
            'content': 'What is the capital of France?',
        },
    ],
    'max_tokens': 500,
    'stream': True,
}

response = requests.post(
    'https://api-inference.huggingface.co/models/google/gemma-2-2b-it/v1/chat/completions',
    headers=headers,
    json=json_data,
    verify=False
)

print(response.text)
