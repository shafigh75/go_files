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


# sample image generation: 

import requests

API_URL = "https://api-inference.huggingface.co/models/black-forest-labs/FLUX.1-dev"
headers = {"Authorization": "Bearer hf_***"}

def query(payload):
	response = requests.post(API_URL, headers=headers, json=payload)
	return response.content
image_bytes = query({
	"inputs": "Astronaut riding a horse",
})

# You can access the image with PIL.Image for example
import io
from PIL import Image
image = Image.open(io.BytesIO(image_bytes))
