from flask import Flask, request, jsonify, render_template, Response
from flask_cors import CORS
import json
from zai import ZaiClient
import os

app = Flask(__name__)
CORS(app)

# Store conversation history in memory (in production, use a database)
conversations = {}

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/chat', methods=['POST'])
def chat():
    data = request.json
    user_id = data.get('user_id', 'default_user')
    message = data.get('message', '')
    thinking_mode = data.get('thinking_mode', True)

    # Initialize conversation history if not exists
    if user_id not in conversations:
        conversations[user_id] = []

    # Add user message to history
    conversations[user_id].append({"role": "user", "content": message})

    # Prepare API request
    client = ZaiClient(api_key="ebb378fb1657450583f64e6d0e94636b.5c66cIYnwtcNRNRL")

    # Create thinking parameter based on user preference
    thinking_param = {"type": "enabled"} if thinking_mode else {"type": "disabled"}

    # Generate response
    def generate():
        response = client.chat.completions.create(
            model="glm-4.5-flash",
            messages=conversations[user_id],
            thinking=thinking_param,
            max_tokens=4096,
            temperature=0.6,
            stream=True  # Enable streaming
        )

        full_response = ""
        for chunk in response:
            if chunk.choices[0].delta.content:
                content = chunk.choices[0].delta.content
                full_response += content
                yield f"data: {json.dumps({'content': content})}\n\n"

        # Add assistant response to history
        conversations[user_id].append({"role": "assistant", "content": full_response})

    return Response(generate(), mimetype='text/event-stream')

@app.route('/clear', methods=['POST'])
def clear_conversation():
    data = request.json
    user_id = data.get('user_id', 'default_user')
    if user_id in conversations:
        conversations[user_id] = []
    return jsonify({"status": "success"})

if __name__ == '__main__':
    # Get port from environment variable or use default
    port = int(os.environ.get('PORT', 5000))

    # Run on all interfaces
    app.run(host='0.0.0.0', port=port, debug=True)
