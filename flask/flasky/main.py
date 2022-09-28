from flask import Flask

app = Flask(__name__)


@app.route('/')
def index():
    return '<h1>Ciao Mondo</h1>'


@app.route('/user/<name>')
def user(name):
    return '<h1>Ciao {}</h1>'.format(name)


"""
export FLASK_APP=main.py
(flask) [vcrini@Fenice] ➜ flasky (! master) flask run
"""
