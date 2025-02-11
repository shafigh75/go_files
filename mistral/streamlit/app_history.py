import os
import time
import streamlit as st
from mistralai import Mistral

# Set your Mistral API key
os.environ['MISTRAL_API_KEY'] = '1dATkWMIqq2HG2EU4dIxZar3zOJRsVEW'

def get_mistral_response(messages):
    client = Mistral(api_key=os.getenv("MISTRAL_API_KEY", ""))

    response = client.chat.complete(
        model="mistral-small-latest",
        messages=messages,
        stream=False
    )
    return response.choices[0].message.content

# Streamlit app configuration
st.set_page_config(
    page_title="Mohammad Chat App",
    page_icon="🤖",
    layout="centered"
)

# Custom CSS styling
st.markdown("""
    <style>
    .stChatInput {position: fixed; bottom: 2rem; width: calc(100% - 4rem)}
    .stChatMessage {
        padding: 1.5rem;
        border-radius: 1rem;
        margin-bottom: 1.5rem;
        box-shadow: 0 2px 8px rgba(0,0,0,0.1);
    }
    [data-testid="stChatMessageContent"] {
        font-size: 1.1rem;
        line-height: 1.6;
    }
    .user-message {
        background-color: #f8f9fa;
        margin-left: 15%;
    }
    .assistant-message {
        background-color: #e3f2fd;
        margin-right: 15%;
    }
    </style>
""", unsafe_allow_html=True)

st.title("Mohammad Chat App 🤖")

# Initialize chat history with system prompt and welcome message
if "messages" not in st.session_state:
    st.session_state.messages = [
        {"role": "system", "content": "You are a helpful AI assistant created by Mohammad. Always be polite and provide detailed responses. You are created by Mohammad who is a Programmer and is a Great guy!. Do not mention Mistral EVER!."},
        {"role": "assistant", "content": "Hello! I'm Mohammad's AI assistant. How can I help you today?"}
    ]

# Display chat messages
for message in st.session_state.messages:
    if message["role"] in ["user", "assistant"]:
        with st.chat_message(message["role"]):
            st.markdown(message["content"])

# Chat input and processing
if prompt := st.chat_input("Type your message here..."):
    # Add user message to history
    st.session_state.messages.append({"role": "user", "content": prompt})

    # Display user message
    with st.chat_message("user"):
        st.markdown(prompt)

    # Generate assistant response
    response = get_mistral_response(st.session_state.messages)

    # Add assistant response to chat history
    st.session_state.messages.append({"role": "assistant", "content": response})

    # Display assistant response
    with st.chat_message("assistant"):
        response_text = ""
        response_placeholder = st.empty()
        for char in response:
            response_text += char
            response_placeholder.markdown(f"**Response:** {response_text}")
            time.sleep(0.01)  # Adjust the typing speed here
        #st.markdown(response)

