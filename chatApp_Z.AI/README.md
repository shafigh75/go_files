

# AI Chat Application

A real-time streaming chat application built with Flask and the ZAI API, featuring conversation memory, thinking mode toggle, and Markdown rendering for responses.

## Features

- Real-time streaming responses for a ChatGPT-like experience
- Conversation memory to maintain context across messages
- Toggle for thinking mode (enabled/disabled)
- Markdown rendering for formatted responses (code blocks, lists, links, etc.)
- Syntax highlighting for code snippets
- Responsive design that works on all devices
- Accessible on all network interfaces

## Quickstart Guide

Here's a simple implementation to get started with the ZAI API:

```python
from zai import ZaiClient

client = ZaiClient(api_key="ebb378fb1657450583f64e6d0e94636b.5c66cIYnwtcNRNRL")  # Your API Key

response = client.chat.completions.create(
    model="glm-4.5-flash",
    messages=[
        {"role": "user", "content": "As a marketing expert, please create an attractive slogan for my product."},
        {"role": "assistant", "content": "Sure, to craft a compelling slogan, please tell me more about your product."},
        {"role": "user", "content": "a web store which sells all kind of stuff to a cheap price in UK and we focus on finding best deals"}
    ],
    thinking={
        "type": "enabled",
    },
    max_tokens=4096,
    temperature=0.6
)

# Get complete response
print(response.choices[0].message)
```

## Installation

1. Clone this repository:
```bash
git clone https://github.com/yourusername/ai-chat-app.git
cd ai-chat-app
```

2. Create a virtual environment (recommended):
```bash
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
```

3. Install the required packages:
```bash
pip install flask flask-cors zai
```

## Usage

### Development Mode

1. Run the Flask application:
```bash
python app.py
```

2. Open your browser and navigate to:
   - Local access: `http://localhost:5000`
   - Network access: `http://YOUR_IP_ADDRESS:5000` (replace with your machine's IP)

3. Start chatting with the AI assistant!

### Production Mode

For production deployment, it's recommended to use a production-grade WSGI server like Gunicorn or Uvicorn. Here's how to set it up:

#### Using Gunicorn

1. Install Gunicorn:
```bash
pip install gunicorn
```

2. Run the application with Gunicorn:
```bash
gunicorn --bind 0.0.0.0:5000 --workers 4 app:app
```

   - `--bind 0.0.0.0:5000`: Makes the server accessible on all network interfaces on port 5000
   - `--workers 4`: Sets the number of worker processes (adjust based on your server's CPU cores)

3. For better performance, consider using gevent workers:
```bash
pip install gevent
gunicorn --bind 0.0.0.0:5000 --worker-class gevent --workers 4 app:app
```

#### Using Uvicorn

1. Install Uvicorn:
```bash
pip install uvicorn
```

2. Run the application with Uvicorn:
```bash
uvicorn app:app --host 0.0.0.0 --port 5000 --workers 4
```

   - `--host 0.0.0.0`: Makes the server accessible on all network interfaces
   - `--port 5000`: Sets the port to 5000
   - `--workers 4`: Sets the number of worker processes

#### Using Uvicorn with Gunicorn (for ASGI support)

If you want to use Uvicorn workers within Gunicorn:

1. Install the required packages:
```bash
pip install uvicorn gunicorn
```

2. Run the application:
```bash
gunicorn --bind 0.0.0.0:5000 --worker-class uvicorn.workers.UvicornWorker --workers 4 app:app
```

### Configuration

- The application runs on all network interfaces by default (`host='0.0.0.0'`)
- To use a custom port:
  ```bash
  PORT=8080 python app.py  # Development mode
  gunicorn --bind 0.0.0.0:8080 app:app  # Production with Gunicorn
  uvicorn app:app --host 0.0.0.0 --port 8080  # Production with Uvicorn
  ```

## Model Information

**Important**: Currently, only the `glm-4.5-flash` model is available for free use. Other models may require payment or have usage restrictions.

## Project Structure

```
ai-chat-app/
├── app.py                 # Flask backend
└── templates/
    └── index.html         # Frontend HTML with JavaScript
```

## API Reference

### Endpoints

- `GET /`: Returns the main chat interface
- `POST /chat`: Sends a message to the AI and streams the response
  - Request Body:
    ```json
    {
      "user_id": "unique_user_id",
      "message": "Your message here",
      "thinking_mode": true
    }
    ```
- `POST /clear`: Clears the conversation history for a user
  - Request Body:
    ```json
    {
      "user_id": "unique_user_id"
    }
    ```

## Security Considerations

For production use:

1. Move the API key to environment variables instead of hardcoding it:
   ```python
   import os
   api_key = os.environ.get("ZAI_API_KEY")
   client = ZaiClient(api_key=api_key)
   ```

2. Implement proper user authentication
3. Use a database for conversation storage instead of in-memory storage
4. Set up rate limiting to prevent abuse
5. Use HTTPS instead of HTTP (consider using a reverse proxy like Nginx)
6. Configure your firewall properly
7. Use a process manager like systemd or supervisor to keep the application running

## License

This project is licensed under the MIT License.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

If you encounter any issues or have questions, please open an issue on GitHub.
