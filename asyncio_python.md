# python asyncio

`asyncio` is a library in Python for writing concurrent code using the async/await syntax. It is particularly useful for I/O-bound and high-level structured network code. Here's a comprehensive guide to understanding `asyncio`.

## Overview of `asyncio`

`asyncio` is designed to handle asynchronous I/O operations. It allows you to write code that performs non-blocking operations while waiting for I/O, network responses, or other tasks.

### Key Concepts

1. **Event Loop**: The core component of `asyncio`. It runs asynchronous tasks and callbacks, handles I/O, and manages all the event-driven operations.
2. **Coroutines**: Special functions defined with `async def` that use `await` to pause execution until the awaited task completes.
3. **Tasks**: A way to schedule coroutines to run concurrently.
4. **Futures**: Objects that represent a result that may not have been computed yet.
5. **Executors**: Used for running synchronous code in a separate thread or process.

### Basic Usage

#### 1. Creating an Event Loop

You generally don’t need to create an event loop manually; `asyncio.run()` handles it for you. However, you can create one explicitly:

```python
import asyncio

loop = asyncio.get_event_loop()
```

#### 2. Defining Coroutines

Coroutines are defined using `async def` and use `await` to pause execution:

```python
async def my_coroutine():
    print("Start")
    await asyncio.sleep(1)
    print("End")
```

#### 3. Running Coroutines

To run a coroutine, you need to use an event loop:

```python
asyncio.run(my_coroutine())
```

### Scheduling Tasks

#### 1. Creating Tasks

Tasks are used to run coroutines concurrently:

```python
async def my_coroutine():
    await asyncio.sleep(1)
    print("Hello")

# Create a task
task = asyncio.create_task(my_coroutine())
await task
```

#### 2. Gathering Results

You can run multiple coroutines concurrently and gather their results:

```python
async def coro1():
    await asyncio.sleep(1)
    return "Result 1"

async def coro2():
    await asyncio.sleep(2)
    return "Result 2"

results = await asyncio.gather(coro1(), coro2())
print(results)  # Output: ['Result 1', 'Result 2']
```

### Handling I/O Operations

#### 1. Asynchronous I/O

`asyncio` provides non-blocking I/O operations:

```python
async def fetch_data():
    await asyncio.sleep(2)
    return "Data"

data = await fetch_data()
print(data)
```

#### 2. Network Operations

For network operations, you can use `asyncio` with libraries like `aiohttp`:

```python
import aiohttp

async def fetch(url):
    async with aiohttp.ClientSession() as session:
        async with session.get(url) as response:
            return await response.text()

html = await fetch('http://example.com')
print(html)
```

### Synchronization Primitives

`asyncio` provides several primitives for synchronization:

#### 1. Locks

Locks prevent multiple coroutines from accessing a critical section of code simultaneously:

```python
lock = asyncio.Lock()

async def critical_section():
    async with lock:
        # Critical section code
        await asyncio.sleep(1)
```

#### 2. Events

Events are used to notify coroutines about changes in state:

```python
event = asyncio.Event()

async def wait_for_event():
    print("Waiting for event...")
    await event.wait()
    print("Event triggered!")

async def trigger_event():
    await asyncio.sleep(2)
    event.set()

await asyncio.gather(wait_for_event(), trigger_event())
```

### Executors

Executors allow you to run blocking code in a separate thread or process:

```python
from concurrent.futures import ThreadPoolExecutor

def blocking_task(n):
    import time
    time.sleep(n)
    return "Done"

async def run_blocking_task():
    loop = asyncio.get_event_loop()
    with ThreadPoolExecutor() as executor:
        result = await loop.run_in_executor(executor, blocking_task, 3)
    print(result)

await run_blocking_task()
```

### Error Handling

Handle exceptions in coroutines using standard try/except blocks:

```python
async def my_coroutine():
    try:
        await asyncio.sleep(1)
        raise ValueError("An error occurred")
    except ValueError as e:
        print(f"Error: {e}")

await my_coroutine()
```

### Practical Example

Here’s a practical example that combines multiple features:

```python
import asyncio
from concurrent.futures import ThreadPoolExecutor

async def async_task(name):
    print(f"Async task {name} started")
    await asyncio.sleep(2)
    print(f"Async task {name} finished")

def sync_task(name):
    print(f"Sync task {name} started")
    import time
    time.sleep(3)
    print(f"Sync task {name} finished")

async def run_sync_task(name):
    loop = asyncio.get_event_loop()
    with ThreadPoolExecutor() as executor:
        await loop.run_in_executor(executor, sync_task, name)

async def main():
    # Start async tasks
    task1 = asyncio.create_task(async_task("1"))
    task2 = asyncio.create_task(async_task("2"))

    # Start sync task
    await run_sync_task("sync")

    # Await async tasks
    await task1
    await task2

asyncio.run(main())
```

### Summary

- **Event Loop**: Manages all asynchronous operations.
- **Coroutines**: Use `async def` and `await` for non-blocking operations.
- **Tasks**: Schedule coroutines to run concurrently.
- **Futures**: Represent results that are not yet available.
- **Executors**: Run blocking code in separate threads or processes.
- **Synchronization**: Use locks, events, and semaphores to manage concurrency.

Understanding `asyncio` will allow you to write efficient, non-blocking code for I/O-bound tasks and network operations. Experiment with these concepts to get a deeper grasp of asynchronous programming in Python.
