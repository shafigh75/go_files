# agent_rag.py
from pydantic_ai import Agent, Tool
from pydantic import BaseModel
from sentence_transformers import SentenceTransformer
import numpy as np
import datetime
import requests
import json
import re

# -----------------------------
# 1. Local embedding model
# -----------------------------
embed_model = SentenceTransformer("all-MiniLM-L6-v2")
class SimpleVectorStore:
    def __init__(self):
        self.docs = []
        self.embeddings = []
    def add_document(self, text):
        emb = embed_model.encode(text)
        self.docs.append(text)
        self.embeddings.append(emb)
    def similarity_search(self, query, top_k=2):
        q_emb = embed_model.encode(query)
        sims = [np.dot(q_emb, e)/(np.linalg.norm(q_emb)*np.linalg.norm(e)) for e in self.embeddings]
        top_indices = sorted(range(len(sims)), key=lambda i: sims[i], reverse=True)[:top_k]
        return [self.docs[i] for i in top_indices if sims[i] > 0.1]

# -----------------------------
# 2. Prepare FAQ vector store
# -----------------------------
faq_texts = [
    # Previous 3 FAQs
    "We offer standard (5-7 days) and express (2-3 days) shipping options.",
    "You can return any item within 14 days from delivery, no questions asked.",
    "We ship to over 40 countries worldwide. Check the checkout for available countries.",
    # 20+ new FAQs
    "Orders can be cancelled within 24 hours of placement.",
    "We accept payments via credit card, debit card, PayPal, and gift cards.",
    "You can track your order using the tracking number sent in your confirmation email.",
    "If a product is out of stock, we notify customers immediately via email.",
    "We offer a 1-year warranty on all electronic products.",
    "Damaged items can be returned and replaced free of charge within 14 days.",
    "Refunds are processed within 5-7 business days after receiving the returned item.",
    "We offer gift wrapping options during checkout for special occasions.",
    "Customer support is available 24/7 via chat and email.",
    "You can update your shipping address before the order is shipped.",
    "We occasionally run seasonal promotions and discount codes.",
    "Subscriptions can be paused or cancelled anytime from your account settings.",
    "We follow GDPR guidelines to protect customer data and privacy.",
    "Some products have size charts to help you select the correct fit.",
    "For bulk orders, please contact our sales team for special pricing.",
    "We offer eco-friendly packaging options for all orders.",
    "Out-of-delivery-area orders may incur additional shipping charges.",
    "Backordered items are shipped as soon as they become available.",
    "International shipping duties and taxes are the responsibility of the customer.",
    "You can create an account to save addresses and payment details for faster checkout.",
    "Pre-orders are charged at the time of shipping, not when ordered.",
    "ما به همه کد تخفیف میدهیم فقط کافیست در خبرناه ما ثبتنام کنید"
]

faq_store = SimpleVectorStore()
for text in faq_texts:
    faq_store.add_document(text)

# -----------------------------
# 3. RAG retrieval tool
# -----------------------------
def rag_faq(query: str, top_k: int = 2) -> str:
    results = faq_store.similarity_search(query, top_k)
    return "\n".join(results) if results else "No relevant FAQ found."

# -----------------------------
# 4. Order database tool
# -----------------------------
ORDERS_DB = {
    "123": {"status": "shipped", "date": "2025-08-28", "items": ["Laptop", "Mouse"]},
    "456": {"status": "broken", "date": "2025-09-01", "items": ["Keyboard"]},
    "789": {"status": "delivered", "date": "2025-08-20", "items": ["Monitor", "HDMI Cable"]},
}
def check_order(order_id: str) -> str:
    order = ORDERS_DB.get(order_id)
    if not order:
        return f"Order {order_id} not found."
    items = ", ".join(order["items"])
    return f"Order {order_id} is {order['status']} (since {order['date']}), containing items: {items}"

# -----------------------------
# 5. Pydantic structured output
# -----------------------------
class Answer(BaseModel):
    summary: str
    tool_used: str = ""  # Changed default to empty string instead of None
    timestamp: str = datetime.datetime.now().isoformat()

# -----------------------------
# 6. Custom ZAI Model Adapter
# -----------------------------
class ZAIModel:
    def __init__(self, api_key: str, model_name: str = "glm-4.5-flash"):
        self.api_key = api_key
        self.model_name = model_name
        self.base_url = "https://open.bigmodel.cn/api/paas/v4"  # ZAI API endpoint

    def chat_completions_create(self, messages, **kwargs):
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json"
        }

        payload = {
            "model": self.model_name,
            "messages": messages,
            "stream": False
        }

        # Add optional parameters
        if "temperature" in kwargs:
            payload["temperature"] = kwargs["temperature"]
        if "max_tokens" in kwargs:
            payload["max_tokens"] = kwargs["max_tokens"]

        response = requests.post(
            f"{self.base_url}/chat/completions",
            headers=headers,
            json=payload
        )

        if response.status_code != 200:
            raise Exception(f"ZAI API Error: {response.status_code} - {response.text}")

        return response.json()

    def generate(self, system_prompt, messages, tools=None):
        # Format messages for ZAI
        formatted_messages = [{"role": "system", "content": system_prompt}]

        # Add conversation history
        for msg in messages:
            formatted_messages.append({"role": "user", "content": msg})

        # If tools are provided, we need to handle them differently
        if tools:
            # For simplicity, we'll include tool information in the system prompt
            tool_descriptions = "\n".join([f"{tool.name}: {tool.description}" for tool in tools])
            formatted_messages[0]["content"] += f"\n\nAvailable tools:\n{tool_descriptions}"

        response = self.chat_completions_create(formatted_messages)

        # Extract the content from the response
        if "choices" in response and len(response["choices"]) > 0:
            return response["choices"][0]["message"]["content"]
        else:
            raise Exception("Invalid response from ZAI API")

# -----------------------------
# 7. Initialize ZAI agent
# -----------------------------
zai_model = ZAIModel(api_key="ebb378fb1657450583f64e6d0e94636b.5c66cIYnwtcNRNRL")

class ZAIAgent:
    def __init__(self, model, system_prompt, tools=None):
        self.model = model
        self.system_prompt = system_prompt
        self.tools = tools or {}
        self.tool_functions = {tool.name: tool.function for tool in tools} if tools else {}

    def run_sync(self, query):
        # First, check if we should use a tool
        tool_result = None
        tool_used = ""

        # Improved tool detection
        # Check for order-related queries
        order_match = re.search(r'order\s*(\d+)', query.lower())
        if order_match:
            order_id = order_match.group(1)
            if "check_order" in self.tool_functions:
                tool_result = self.tool_functions["check_order"](order_id)
                tool_used = "check_order"
        # Check for FAQ-related queries
        elif any(keyword in query.lower() for keyword in ["shipping", "return", "policy", "faq", "discount", "کد تخفیف"]):
            if "rag_faq" in self.tool_functions:
                tool_result = self.tool_functions["rag_faq"](query)
                tool_used = "rag_faq"

        # If we used a tool, incorporate the result
        if tool_result:
            # Create a prompt that includes the tool result
            enhanced_prompt = f"""
            {self.system_prompt}

            User query: {query}

            Information from {tool_used} tool: {tool_result}

            Please provide a helpful response based on this information.
            """

            # Generate response using ZAI with the enhanced prompt
            response = self.model.generate(
                system_prompt="",  # System prompt is included in the user message
                messages=[enhanced_prompt],
                tools=self.tools
            )
        else:
            # No tool used, just generate a response
            response = self.model.generate(
                system_prompt=self.system_prompt,
                messages=[query],
                tools=self.tools
            )

        # Create a result object similar to Pydantic AI's result
        class Result:
            def __init__(self, output, tool_used=""):
                self.output = output
                self.tool_used = tool_used

        return Result(response, tool_used)

agent = ZAIAgent(
    model=zai_model,
    system_prompt=(
        "You are a helpful customer support assistant. "
        "Always use tools to answer factual questions accurately. "
        "Use 'check_order' for order queries. Use 'rag_faq' for FAQ queries. "
        "Do not hallucinate information. If you don't know the answer to a question, just say it is not registered."
    ),
    tools=[
        Tool(
            name="check_order",
            description="Check the status of a customer order by order ID",
            function=check_order
        ),
        Tool(
            name="rag_faq",
            description="Fetch top FAQ answers from company knowledge base",
            function=rag_faq
        )
    ]
)

# -----------------------------
# 8. Test queries
# -----------------------------
test_queries = [
    "Can you tell me the status of order 456?",
    "How long is your return policy?",
    "Do you ship internationally?",
    "What shipping options do you offer?",
    "can i update my shipping address?",
    "آیا شما کد تخفیف هم دارید؟",
]

for query in test_queries:
    result = agent.run_sync(query)
    answer = Answer(summary=result.output, tool_used=result.tool_used)
    print("\nCustomer:", query)
    print("AI Response:", answer.summary)
    print("Tool Used:", answer.tool_used)
    print("Timestamp:", answer.timestamp)
