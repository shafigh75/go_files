# agent_rag.py
from pydantic_ai import Agent, Tool
from pydantic_ai.models.mistral import MistralModel
from pydantic import BaseModel
from sentence_transformers import SentenceTransformer
import numpy as np
import datetime

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
#faq_texts = [
#    "We offer standard (5-7 days) and express (2-3 days) shipping options.",
#    "You can return any item within 14 days from delivery, no questions asked.",
#    "We ship to over 40 countries worldwide. Check the checkout for available countries.",
#]

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
    "Pre-orders are charged at the time of shipping, not when ordered."
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
    tool_used: str = None
    timestamp: str = datetime.datetime.now().isoformat()

# -----------------------------
# 6. Initialize Mistral agent
# -----------------------------
mistral_model = MistralModel("mistral-large-latest")

agent = Agent(
    model=mistral_model,
    system_prompt=(
        "You are a helpful customer support assistant. "
        "Always use tools to answer factual questions accurately. "
        "Use 'check_order' for order queries. Use 'rag_faq' for FAQ queries. "
        "Do not hallucinate information. if you don't know answer to a question just say it is not registered"
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
# 7. Test queries
# -----------------------------
test_queries = [
    "Can you tell me the status of order 456?",
    "How long is your return policy?",
    "Do you ship internationally?",
    "What shipping options do you offer?",
    "can i update my shipping address?",
]

for query in test_queries:
    result = agent.run_sync(query)
    answer = Answer(summary=result.output)
    print("\nCustomer:", query)
    print("AI Response:", answer.summary)
    print("Timestamp:", answer.timestamp)
