
# What is LangChain?

LangChain is a robust framework designed for building applications that leverage Large Language Models (LLMs), such as GPT-4 or other AI models. It streamlines tasks like generating responses, retrieving data, and handling workflows. Its chainable structure enables complex interactions by chaining together multiple steps.



## Key Features of LangChain

- **Chains**: Build workflows that pass data through several steps.
- **Prompt Templates**: Standardize and dynamically create prompts.
- **Memory**: Track interactions to create conversational agents that "remember".
- **Document Loaders**: Handle unstructured data from files, websites, databases, etc.
- **Vector Stores**: Integrate with vector databases for contextual data retrieval.
- **Tools**: Enhance LLM capabilities with external tools like APIs or Python functions.

---

## Installation

First, install LangChain and any necessary dependencies:

```bash
pip install langchain openai python-dotenv
```

You might also need additional libraries based on the components you use, such as `pandas`, `faiss`, `chromadb`, or `httpx`.

---

## Setting Up OpenAI API Key

1. Grab an API key from [OpenAI's dashboard](https://platform.openai.com/).
2. Store the key securely in a `.env` file:

```env
OPENAI_API_KEY=your_api_key_here
```

3. Load the API key in Python:

```python
import os
from dotenv import load_dotenv

load_dotenv()
OPENAI_API_KEY = os.getenv('OPENAI_API_KEY')
```

---

## Step-by-Step Practical Tutorial

### 1. Basic Language Model Response

Use an LLM like GPT for response generation:

```python
from langchain.chat_models import ChatOpenAI

# Create an OpenAI chat model object
llm = ChatOpenAI(model_name="gpt-4", openai_api_key=OPENAI_API_KEY)

# Simple prompt example
response = llm.predict("Tell me a joke about computers")
print(response)
```

---

### 2. Chains

Chains are workflows where the output from one step becomes input for the next.

#### Using `LLMChain` with Prompt Templates

```python
from langchain.prompts import PromptTemplate
from langchain.chains import LLMChain

# Define a prompt template
template = "What is a good name for a company that makes {product}?"
prompt = PromptTemplate(input_variables=["product"], template=template)

# Create a chain using the prompt and LLM
chain = LLMChain(llm=llm, prompt=prompt)

# Run the chain
product_name = chain.run("artificial intelligence tools")
print(product_name)
```

---

### 3. Tools and Agents

Agents use LLMs to decide which tools to use and what actions to take.

#### Using Tools (e.g., Wikipedia)

```python
from langchain.agents import initialize_agent, Tool
from langchain.utilities import WikipediaAPIWrapper

# Tool: Wikipedia search
wiki = WikipediaAPIWrapper()
tools = [
    Tool(
        name="Wiki Search",
        func=wiki.run,
        description="Search Wikipedia for factual information."
    )
]

# Initialize the agent
agent = initialize_agent(
    tools,
    llm,
    agent_type="zero-shot-react-description",
    verbose=True
)

# Ask the agent a question
agent_response = agent.run("Tell me about the history of Python programming language.")
print(agent_response)
```

---

### 4. Memory

Memory allows LangChain to retain context across interactions.

#### Adding Conversation Memory

```python
from langchain.chains import ConversationChain
from langchain.memory import ConversationBufferMemory

# Initialize memory
memory = ConversationBufferMemory()

# Create the conversational agent
conversation = ConversationChain(llm=llm, memory=memory, verbose=True)

# Start a conversation
conversation.run("Hi, my name is Alex")
conversation.run("What do you think about AI?")
conversation.run("Do you remember my name?")
```

> The output will demonstrate the agent recalling your name based on memory.

---

### 5. Document Retrieval and Question Answering

LangChain enables loading, indexing, and retrieving data for QA systems.

#### Loading Documents

```python
from langchain.document_loaders import TextLoader

loader = TextLoader("example.txt")
documents = loader.load()

print(documents[0].page_content)  # Preview content
```

#### Embedding Data for Contextual Retrieval

```python
from langchain.text_splitter import CharacterTextSplitter
from langchain.vectorstores import FAISS
from langchain.embeddings.openai import OpenAIEmbeddings

# Split documents
text_splitter = CharacterTextSplitter(chunk_size=500, chunk_overlap=0)
texts = text_splitter.split_documents(documents)

# Embed and store
embeddings = OpenAIEmbeddings(openai_api_key=os.getenv("OPENAI_API_KEY"))
vectorstore = FAISS.from_documents(texts, embeddings)

# Query
query = "What is the main topic discussed in this text?"
search_results = vectorstore.similarity_search(query)
print(search_results[0].page_content)
```

#### Combining Retrieval with LLM Question Answering

```python
from langchain.chains import RetrievalQA

qa_chain = RetrievalQA.from_chain_type(
    llm=llm,
    retriever=vectorstore.as_retriever(),
    return_source_documents=True
)

query_result = qa_chain.run("Summarize the text for me.")
print(query_result)
```

---

### 6. API Integration

LangChain can interact with external APIs and combine results with LLM logic.

#### Calling APIs (Example: Weather)

```python
import requests

def weather_api(location):
    response = requests.get(
        f"http://api.weatherapi.com/v1/current.json?key=your_weather_api_key&q={location}"
    )
    return response.json()

weather_data = weather_api("New York")
print(weather_data["current"])
```

> You can extend `weather_api` as a custom **Tool** in LangChain.

#### Advanced Use Case: Chat Assistant with Tools and Memory

```python
from langchain.agents import initialize_agent, Tool
from langchain.memory import ConversationBufferMemory

tools = [
    Tool(name="weather", func=weather_api, description="Get current weather info.")
]

memory = ConversationBufferMemory()

agent = initialize_agent(
    tools,
    llm,
    agent_type="zero-shot-react-description",
    memory=memory,
    verbose=True
)

response = agent.run("What's the weather in London?")
print(response)
```

---

## Best Practices

- **Start simple**: Experiment with basic LLM and chains.
- **Explore memory**: Use conversation memory for dynamic assistants.
- **Combine tools**: Create more capability-rich workflows.
- **Use vector databases**: Enhance contextual retrieval from large datasets.
- **Test and iterate**: Identify edge cases where workflows may break.

---

## Commonly Used Components

| Component | Purpose |
|----------|--------|
| `PromptTemplate` | Create reusable prompt templates for workflows |
| `LLMChain` | Chain LLMs for multi-step tasks |
| `Agents` | Automate workflows using external tools |
| `Memory` | Store user interactions for continuity |
| `Document Loaders` & `Vector Stores` | Handle unstructured data and enable efficient search |

---

LangChain unlocks the full power of language models by allowing you to design workflows, interact with external data sources, and customize responses. Whether you're building a chatbot, search assistant, or analysis tool, LangChain provides a flexible and scalable framework to bring your ideas to life!

