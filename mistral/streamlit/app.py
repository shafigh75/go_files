import os
import time
import streamlit as st
from mistralai import Mistral

# Set your Mistral API key
os.environ['MISTRAL_API_KEY'] = '1dATkWMIqq2HG2EU4dIxZar3zOJRsVEW'

def get_mistral_response(message):
    with Mistral(api_key=os.getenv("MISTRAL_API_KEY", "")) as mistral:
        response = mistral.chat.complete(
            model="mistral-small-latest",
            messages=[
                {"role": "user", "content": message}
            ],
            stream=False
        )
    return response.choices[0].message.content

# Streamlit app
st.title("Mistral AI Chat")

user_input = st.text_input("You:", placeholder="Type your message here")

if st.button("Send"):
    if user_input:
        response = get_mistral_response(user_input)
        response_placeholder = st.empty()
        response_text = ""

        for char in response:
            response_text += char
            response_placeholder.markdown(f"**Response:** {response_text}")
            time.sleep(0.01)  # Adjust the typing speed here
    else:
        st.warning("Please enter a message.")
