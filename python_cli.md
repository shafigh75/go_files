# python cli using click:

```python
import click

@click.group()
def cli():
    """A simple CLI example"""
    pass

@click.command()
@click.argument('name')
def greet(name):
    """Greet a person by their name"""
    click.echo(f'Hello, {name}!')

@click.command()
@click.option('--count', default=1, help='Number of greetings.')
@click.argument('name')
def repeat_greet(name, count):
    """Greet a person multiple times"""
    for _ in range(count):
        click.echo(f'Hello, {name}!')

cli.add_command(greet)
cli.add_command(repeat_greet)

if __name__ == '__main__':
    cli()

```
