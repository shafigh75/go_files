from flask import Flask, render_template, request
import os
from mistralai import Mistral

app = Flask(__name__)

@app.route('/')
def home():
    return render_template('index.html')

@app.route('/chat', methods=['POST'])
def chat():
    message = request.form['message']
    response = get_mistral_response(message)
    return render_template('index.html', user_message=message, response=response)

def get_mistral_response(message):
    os.environ['MISTRAL_API_KEY'] = '1dATkWMIqq2HG2EU4dIxZar3zOJRsVEW'
    with Mistral(api_key=os.getenv("MISTRAL_API_KEY", "")) as mistral:
        response = mistral.chat.complete(
            model="mistral-small-latest",
            messages=[
                {"role": "user", "content": message}
            ],
            stream=False
        )
    return response.choices[0].message.content

if __name__ == '__main__':
    app.run(debug=True, port=8080)
