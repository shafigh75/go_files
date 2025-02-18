Object-Oriented Programming (OOP) is a programming paradigm that organizes software design around data (objects) rather than functions and logic. It uses several key principles to provide flexibility, modularity, and reusability in code. Here’s an introduction to the four core principles of OOP along with a brief example for each:

---

1. **Encapsulation**

   Encapsulation involves bundling the data (attributes) and the methods (functions) that operate on the data into a single unit (called an object or class). It also hides the internal state of the object and requires all interaction to be performed through its methods. This protects the integrity of the data and prevents unintended interference.

   *Example in Python:*
   
   ```python
   class BankAccount:
       def __init__(self, owner, balance=0):
           self.owner = owner
           self.__balance = balance  # Private attribute
   
       def deposit(self, amount):
           if amount > 0:
               self.__balance += amount
               print(f"Deposited {amount}. New balance: {self.__balance}")
   
       def withdraw(self, amount):
           if 0 < amount <= self.__balance:
               self.__balance -= amount
               print(f"Withdrew {amount}. New balance: {self.__balance}")
           else:
               print("Insufficient funds or invalid amount")
       
       def get_balance(self):
           return self.__balance
   
   # Using the encapsulated class
   account = BankAccount("Alice", 100)
   account.deposit(50)
   account.withdraw(30)
   print("Balance:", account.get_balance())
   ```
   
   In this example, the balance is encapsulated (made private) to prevent direct modification from outside the class.

---

2. **Abstraction**

   Abstraction means hiding complex implementation details and showing only the necessary features of an object. It allows the programmer to focus on interactions at a high level without needing to understand the inner workings.

   *Example in Python:*
   
   ```python
   from abc import ABC, abstractmethod
   
   class Animal(ABC):
       @abstractmethod
       def make_sound(self):
           pass
   
   class Dog(Animal):
       def make_sound(self):
           print("Woof!")
   
   class Cat(Animal):
       def make_sound(self):
           print("Meow!")
   
   # Using abstraction
   def animal_sound(animal: Animal):
       animal.make_sound()
   
   dog = Dog()
   cat = Cat()
   
   animal_sound(dog)  # Output: Woof!
   animal_sound(cat)  # Output: Meow!
   ```
   
   Here, the abstract class Animal defines a high-level interface (the make_sound method), and the concrete classes implement the details.

---

3. **Inheritance**

   Inheritance enables a new class (child or subclass) to inherit attributes and methods from an existing class (parent or superclass). This helps in code reusability and establishing a relationship between classes.

   *Example in Python:*
   
   ```python
   class Vehicle:
       def __init__(self, make, model):
           self.make = make
           self.model = model
   
       def display_info(self):
           print(f"Vehicle: {self.make} {self.model}")
   
   class Car(Vehicle):  # Car inherits from Vehicle
       def __init__(self, make, model, car_type):
           super().__init__(make, model)
           self.car_type = car_type
   
       def display_info(self):
           super().display_info()
           print(f"Car Type: {self.car_type}")
   
   # Using inheritance
   my_car = Car("Toyota", "Corolla", "Sedan")
   my_car.display_info()
   ```
   
   In this example, Car inherits from Vehicle and extends its functionality by adding a new attribute and modifying the display_info method.

---

4. **Polymorphism**

   Polymorphism allows objects of different classes to be treated as objects of a common superclass. It enables methods to be used interchangeably despite differences in their underlying classes. The specific method that gets executed depends on the object's class.

   *Example in Python:*
   
   ```python
   class Shape:
       def area(self):
           raise NotImplementedError("Subclasses should implement this method")
   
   class Rectangle(Shape):
       def __init__(self, width, height):
           self.width = width
           self.height = height
       
       def area(self):
           return self.width * self.height
   
   class Circle(Shape):
       def __init__(self, radius):
           self.radius = radius
       
       def area(self):
           import math
           return math.pi * (self.radius ** 2)
   
   # Function that uses polymorphism
   def print_area(shape: Shape):
       print("Area:", shape.area())
   
   rectangle = Rectangle(4, 5)
   circle = Circle(3)
   
   print_area(rectangle)  # Uses Rectangle's area method
   print_area(circle)     # Uses Circle's area method
   ```
   
   Here, the same function print_area works with any Shape instance, regardless of whether it’s a Rectangle or a Circle. Each shape computes its area differently, demonstrating polymorphism.

---

These examples illustrate how OOP's core principles contribute to writing modular, maintainable, and reusable code.
