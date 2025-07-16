# Design Patterns in Python - Simple, Practical, and Memorable Examples

# ====================
# Creational Patterns
# ====================

# 1. Singleton Pattern
# ----------------------
# Ensures a class has only one instance and provides a global point of access.
# Useful when exactly one object is needed to coordinate actions across the system.
#
# EXPLANATION:
# Imagine a logging system, configuration loader, or database connection pool
# — you only want ONE of these to exist throughout your application.
# Singleton ensures only one instance is created and reused everywhere.

class Singleton:
    _instance = None

    def __new__(cls):
        if cls._instance is None:
            print("Creating a new Singleton instance")
            cls._instance = super().__new__(cls)
        return cls._instance

# Usage:
s1 = Singleton()
s2 = Singleton()
assert s1 is s2  # True - both refer to the same instance


# 2. Factory Method Pattern
# --------------------------
# Defines an interface for creating an object but lets subclasses decide which class to instantiate.
# Helps when the exact type of object isn't known until runtime.
#
# EXPLANATION:
# Think of a UI where the user picks an option (dog, cat), and you want to create the correct object
# without hardcoding logic all over the place. The factory centralizes that creation logic.

class Animal:
    def speak(self):
        pass

class Dog(Animal):
    def speak(self):
        return "Woof!"

class Cat(Animal):
    def speak(self):
        return "Meow!"

class AnimalFactory:
    def create_animal(self, animal_type):
        if animal_type == "dog":
            return Dog()
        elif animal_type == "cat":
            return Cat()
        else:
            raise ValueError("Unknown animal type")

# Usage:
factory = AnimalFactory()
animal = factory.create_animal("dog")
print(animal.speak())  # Output: Woof!


# 3. Builder Pattern
# -------------------
# Separates the construction of a complex object from its representation.
# Useful when you have a class with many optional parts or configuration steps.
#
# EXPLANATION:
# Imagine creating a car where you want optional GPS, sunroof, or autopilot.
# Instead of having a giant constructor with tons of parameters,
# you use a builder to assemble the car step-by-step.

class Car:
    def __init__(self):
        self.engine = None
        self.color = None
        self.sunroof = False

    def __str__(self):
        return f"Car(engine={self.engine}, color={self.color}, sunroof={self.sunroof})"

class CarBuilder:
    def __init__(self):
        self.car = Car()

    def set_engine(self, engine):
        self.car.engine = engine
        return self

    def set_color(self, color):
        self.car.color = color
        return self

    def add_sunroof(self):
        self.car.sunroof = True
        return self

    def build(self):
        return self.car

# Usage:
car = CarBuilder().set_engine("V8").set_color("red").add_sunroof().build()
print(car)  # Car(engine=V8, color=red, sunroof=True)


# ====================
# Structural Patterns
# ====================

# 4. Facade Pattern
# ------------------
# Provides a simplified interface to a larger and more complex system.
# Useful when you want to hide complexity and provide a cleaner API.
#
# EXPLANATION:
# Think of a TV remote. Internally it controls volume, power, input, etc.,
# but gives you simple buttons. That’s a facade.

class CPU:
    def freeze(self):
        print("Freezing CPU")

    def jump(self, position):
        print(f"Jumping to {position}")

    def execute(self):
        print("Executing")

class Memory:
    def load(self, position, data):
        print(f"Loading {data} into memory at {position}")

class HardDrive:
    def read(self, lba, size):
        return f"Data from {lba} of size {size}"

class ComputerFacade:
    def __init__(self):
        self.cpu = CPU()
        self.memory = Memory()
        self.hard_drive = HardDrive()

    def start(self):
        self.cpu.freeze()
        data = self.hard_drive.read(100, 50)
        self.memory.load(100, data)
        self.cpu.jump(100)
        self.cpu.execute()

# Usage:
computer = ComputerFacade()
computer.start()


# 5. Adapter Pattern
# -------------------
# Allows incompatible interfaces to work together.
# Converts the interface of a class into another that clients expect.
# 
# EXPLANATION:
# Suppose your system expects every printer to have a method called `print()`.
# However, you have an old printer with a method called `old_print()` instead.
# Rather than modifying the old printer (maybe you can't),
# you create an adapter that wraps the old printer and provides the `print()` interface.

class OldPrinter:
    def old_print(self):
        print("Printing from old printer")

class NewPrinter:
    def print(self):
        print("Printing from new printer")

# Adapter wraps OldPrinter and gives it the modern print() interface.
class PrinterAdapter:
    def __init__(self, old_printer):
        self.old_printer = old_printer

    def print(self):
        # Internally calls old_print, but externally looks like a print method
        self.old_printer.old_print()

# Usage:
old = OldPrinter()
adapted = PrinterAdapter(old)
adapted.print()  # Output: Printing from old printer


# 6. Decorator Pattern
# ----------------------
# Adds behavior to an object dynamically without changing its code.
#
# EXPLANATION:
# Think of wrapping a function with logging or authentication logic.
# Decorators allow you to enhance behavior without modifying original code.

class Coffee:
    def cost(self):
        return 5

class MilkDecorator:
    def __init__(self, coffee):
        self.coffee = coffee

    def cost(self):
        return self.coffee.cost() + 2

class SugarDecorator:
    def __init__(self, coffee):
        self.coffee = coffee

    def cost(self):
        return self.coffee.cost() + 1

# Usage:
basic = Coffee()
milk_coffee = MilkDecorator(basic)
sweet_milk_coffee = SugarDecorator(milk_coffee)
print("Cost:", sweet_milk_coffee.cost())  # Output: 8


# =====================
# Behavioral Patterns
# =====================

# 7. Observer Pattern
# --------------------
# Defines a one-to-many dependency between objects.
# When one object (the subject) changes, all its observers are notified.
# Useful for event systems, notifications, etc.
#
# EXPLANATION:
# Imagine a news feed or stock ticker. When the data changes,
# all the subscribed observers (users/interfaces) are notified automatically.

class Subject:
    def __init__(self):
        self._observers = []

    def attach(self, observer):
        self._observers.append(observer)

    def detach(self, observer):
        self._observers.remove(observer)

    def notify(self, message):
        for observer in self._observers:
            observer.update(message)

class ConcreteObserver:
    def __init__(self, name):
        self.name = name

    def update(self, message):
        print(f"{self.name} received: {message}")

# Usage:
subject = Subject()
o1 = ConcreteObserver("Observer 1")
o2 = ConcreteObserver("Observer 2")
subject.attach(o1)
subject.attach(o2)
subject.notify("Event occurred")


# 8. Strategy Pattern
# --------------------
# Defines a family of interchangeable algorithms.
# Useful when you want to choose an algorithm at runtime.
#
# EXPLANATION:
# You have multiple ways to perform a task (e.g. add, subtract),
# and you want to swap them easily without changing the surrounding code.
# You encapsulate each algorithm in a separate class and choose which to use.

class Strategy:
    def do_operation(self, a, b):
        pass

class Add(Strategy):
    def do_operation(self, a, b):
        return a + b

class Subtract(Strategy):
    def do_operation(self, a, b):
        return a - b

class Context:
    def __init__(self, strategy):
        self.strategy = strategy

    def execute(self, a, b):
        return self.strategy.do_operation(a, b)

# Usage:
context = Context(Add())
print("Add result:", context.execute(10, 5))  # 15
context = Context(Subtract())
print("Subtract result:", context.execute(10, 5))  # 5
