# file watchers in python:

```python
import time
from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler
import os

class MyHandler(FileSystemEventHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.last_modified = {}

    def is_text_file(self, path):
        try:
            with open(path, 'r', encoding='utf-8'):
                return True
        except (UnicodeDecodeError, IOError):
            return False

    def on_modified(self, event):
        if not event.is_directory:
            current_time = time.time()
            if (event.src_path in self.last_modified and 
                current_time - self.last_modified[event.src_path] < 1):
                return  # Ignore if the event was triggered too recently
            
            self.last_modified[event.src_path] = current_time

            # Skip non-text or temporary files
            if not self.is_text_file(event.src_path):
                return

            print(f'File modified: {event.src_path}')
            try:
                with open(event.src_path, 'r', encoding='utf-8') as file:
                    content = file.read()
                    print(content)
            except (UnicodeDecodeError, IOError) as e:
                print(f"Error reading file {event.src_path}: {e}")

if __name__ == "__main__":
    path = "/tmp/new"  # Directory to watch
    event_handler = MyHandler()
    observer = Observer()
    observer.schedule(event_handler, path, recursive=True)
    observer.start()

    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        observer.stop()
    observer.join()
```
