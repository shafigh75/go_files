```
What Is Event Sourcing?

Event sourcing is a design pattern that stores the state of an application as a sequence of events rather than simply storing the current state. Each event represents a state change and is immutable once recorded. 
Instead of updating a current state in a database (like in CRUD), you append events to an event store. The current state can be reconstructed at any time by replaying these events.

Key Benefits:

    Audit trail: Every change is recorded.
    Reproducibility: You can rebuild the state at any point in time.
    Flexibility: You can derive multiple models (read models, caches etc.) from the event stream.

```

import datetime
from typing import List, Any

# Define our Event Base Class
class Event:
    def __init__(self, occurred_on: datetime.datetime = None):
        self.occurred_on = occurred_on or datetime.datetime.utcnow()

    def __str__(self) -> str:
        return f"{self.__class__.__name__} at {self.occurred_on}"

# Define specific events
class AccountCreated(Event):
    def __init__(self, account_id: str, initial_balance: float, occurred_on: datetime.datetime = None):
        super().__init__(occurred_on)
        self.account_id = account_id
        self.initial_balance = initial_balance

    def __str__(self) -> str:
        return f"{super().__str__()}: Account {self.account_id} created with balance {self.initial_balance}"

class MoneyDeposited(Event):
    def __init__(self, account_id: str, amount: float, occurred_on: datetime.datetime = None):
        super().__init__(occurred_on)
        self.account_id = account_id
        self.amount = amount

    def __str__(self) -> str:
        return f"{super().__str__()}: {self.amount} deposited to account {self.account_id}"

class MoneyWithdrawn(Event):
    def __init__(self, account_id: str, amount: float, occurred_on: datetime.datetime = None):
        super().__init__(occurred_on)
        self.account_id = account_id
        self.amount = amount

    def __str__(self) -> str:
        return f"{super().__str__()}: {self.amount} withdrawn from account {self.account_id}"

# Event store: simple in-memory store
class EventStore:
    def __init__(self):
        self.events = []  # list to store events

    def append(self, event: Event):
        self.events.append(event)
        print(f"Event appended: {event}")

    def get_events(self, account_id: str) -> List[Event]:
        # Select events that belong to a specific account
        return [event for event in self.events if hasattr(event, 'account_id') and event.account_id == account_id]

# Aggregate: BankAccount
class BankAccount:
    def __init__(self, account_id: str):
        self.account_id = account_id
        self.balance = 0.0
        self._uncommitted_events: List[Event] = []

    def create_account(self, initial_balance: float):
        event = AccountCreated(account_id=self.account_id, initial_balance=initial_balance)
        self.apply(event)
        self._uncommitted_events.append(event)

    def deposit(self, amount: float):
        event = MoneyDeposited(account_id=self.account_id, amount=amount)
        self.apply(event)
        self._uncommitted_events.append(event)

    def withdraw(self, amount: float):
        # Optionally, add a check for overdraft etc.
        if self.balance < amount:
            raise ValueError("Insufficient funds")
        event = MoneyWithdrawn(account_id=self.account_id, amount=amount)
        self.apply(event)
        self._uncommitted_events.append(event)

    def apply(self, event: Event):
        # Update the state based on event type
        if isinstance(event, AccountCreated):
            self.balance = event.initial_balance
        elif isinstance(event, MoneyDeposited):
            self.balance += event.amount
        elif isinstance(event, MoneyWithdrawn):
            self.balance -= event.amount

    def load_from_history(self, events: List[Event]):
        for event in events:
            self.apply(event)

    def get_uncommitted_events(self) -> List[Event]:
        return self._uncommitted_events

    def mark_events_as_committed(self):
        self._uncommitted_events = []

    def __str__(self):
        return f"BankAccount({self.account_id}): balance={self.balance}"

# Example usage of event sourcing in action
def main():
    store = EventStore()
    account_id = "ACCT-001"

    # Create a new BankAccount and perform some operations
    account = BankAccount(account_id=account_id)
    account.create_account(initial_balance=100.0)
    account.deposit(50.0)
    try:
        account.withdraw(30.0)
    except ValueError as e:
        print(e)

    # Commit events to the event store
    for event in account.get_uncommitted_events():
        store.append(event)
    account.mark_events_as_committed()

    print(f"Final state of account: {account}")

    # Now, let's simulate rebuilding the account state from the event store
    historical_events = store.get_events(account_id=account_id)
    rebuilt_account = BankAccount(account_id=account_id)
    rebuilt_account.load_from_history(historical_events)
    print(f"Rebuilt account state: {rebuilt_account}")

if __name__ == "__main__":
    main()
