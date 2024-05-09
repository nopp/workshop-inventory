from flask import Flask, render_template, request, jsonify, redirect, url_for
import requests
import pymysql

app = Flask(__name__)

api_url = "http://homer.local/"

db = pymysql.connect(host="homer.local", user="root", passwd="123654",db="homeapp")

@app.route('/')
def index():
    response = requests.get(api_url+"/api/products")
    print(response.status_code)
    products = response.json()
    return render_template('index.html', products=products)

@app.route('/product/<int:id>')
def view_product(id):
    response = requests.get(api_url+"/api/product/"+str(id))
    product = response.json()
    return render_template('product.html', product=product)

@app.route('/product/new', methods=['GET', 'POST'])
def create_product():
    if request.method == 'POST':
        data = {
            'name': request.form['name'],
            'cabinet': request.form['cabinet'],
            'shelf': request.form['shelf']
        }
        print(data)
        response = requests.post(api_url+"/api/product", json=data)
        if response.status_code == 201:
            return redirect('/ws')
    
    cursor = db.cursor()
    cursor.execute("SELECT id, name FROM cabinets")
    cabinets = cursor.fetchall()
    return render_template('create_product.html', cabinets=cabinets)

@app.route('/shelves/<int:cabinet_id>')
def get_shelves(cabinet_id):
    cursor = db.cursor()
    cursor.execute("SELECT id,name,position FROM shelves WHERE cabinet_id = %s", (cabinet_id,))
    shelves = cursor.fetchall()
    return jsonify(shelves)

@app.route('/product/edit/<int:id>', methods=['GET', 'POST'])
def edit_product(id):
    response = requests.get(api_url+"/api/product/"+str(id))
    product = response.json()

    if request.method == 'POST':
        data = {
            'name': request.form['name'],
            'cabinet': request.form['cabinet'],
            'shelf': request.form['shelf']
        }
        response = requests.put(api_url+"/product/"+str(id), json=data)
        if response.status_code == 200:
            return redirect('/')

    return render_template('edit_product.html', product=product)

@app.route('/product/delete/<int:id>', methods=['POST'])
def delete_product(id):
    response = requests.delete(api_url+"/api/product/"+str(id))
    if response.status_code == 204:
        return redirect('/')

if __name__ == '__main__':
    app.run(host="0.0.0.0",port=7676,debug=True)
