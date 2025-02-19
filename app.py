import os
import pymysql
from flask import Flask, render_template, request, redirect, url_for, flash
from flask_sqlalchemy import SQLAlchemy
from flask_login import LoginManager, UserMixin, login_user, login_required, logout_user, current_user
from werkzeug.utils import secure_filename

app = Flask(__name__)
app.config['SQLALCHEMY_DATABASE_URI'] = 'mysql://user:password@db/inventario'
app.config['SECRET_KEY'] = 'chave_secreta'
app.config['UPLOAD_FOLDER'] = 'static/uploads'
os.makedirs(app.config['UPLOAD_FOLDER'], exist_ok=True)
db = SQLAlchemy(app)
login_manager = LoginManager(app)
login_manager.login_view = 'login'

class User(db.Model, UserMixin):
    id = db.Column(db.Integer, primary_key=True)
    username = db.Column(db.String(100), unique=True, nullable=False)
    password = db.Column(db.String(100), nullable=False)

class Peca(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    nome = db.Column(db.String(100), nullable=False)
    descricao = db.Column(db.Text, nullable=True)
    estante = db.Column(db.String(20), nullable=False)
    prateleira = db.Column(db.String(20), nullable=False)
    segmento = db.Column(db.String(20), nullable=False)
    imagem = db.Column(db.String(255), nullable=True)

@login_manager.user_loader
def load_user(user_id):
    return User.query.get(int(user_id))

@app.route('/register', methods=['GET', 'POST'])
def register():
    if request.method == 'POST':
        username = request.form['username']
        password = request.form['password']
        if User.query.filter_by(username=username).first():
            flash('Usuário já existe', 'danger')
            return redirect(url_for('register'))
        user = User(username=username, password=password)
        db.session.add(user)
        db.session.commit()
        flash('Usuário cadastrado com sucesso!', 'success')
        return redirect(url_for('login'))
    return render_template('register.html')

@app.route('/login', methods=['GET', 'POST'])
def login():
    if request.method == 'POST':
        username = request.form['username']
        password = request.form['password']
        user = User.query.filter_by(username=username, password=password).first()
        if user:
            login_user(user)
            return redirect(url_for('index'))
        flash('Login inválido', 'danger')
    return render_template('login.html')

@app.route('/logout')
@login_required
def logout():
    logout_user()
    return redirect(url_for('login'))

@app.route('/', methods=['GET', 'POST'])
@login_required
def index():
    query = request.form.get('query', '')
    if query:
        pecas = Peca.query.filter(Peca.nome.ilike(f"%{query}%")).all()
    else:
        pecas = Peca.query.all()
    return render_template('index.html', pecas=pecas, query=query)

@app.route('/add', methods=['GET', 'POST'])
@login_required
def add():
    if request.method == 'POST':
        nome = request.form['nome']
        descricao = request.form['descricao']
        estante = request.form['estante']
        prateleira = request.form['prateleira']
        segmento = request.form['segmento']
        imagem = request.files['imagem']
        imagem_filename = None
        if imagem:
            imagem_filename = secure_filename(imagem.filename)
            imagem.save(os.path.join(app.config['UPLOAD_FOLDER'], imagem_filename))
        nova_peca = Peca(nome=nome, descricao=descricao, estante=estante, prateleira=prateleira, segmento=segmento, imagem=imagem_filename)
        db.session.add(nova_peca)
        db.session.commit()
        return redirect(url_for('index'))
    return render_template('add.html')

@app.route('/edit/<int:id>', methods=['GET', 'POST'])
@login_required
def edit(id):
    peca = Peca.query.get(id)
    if request.method == 'POST':
        peca.nome = request.form['nome']
        peca.descricao = request.form['descricao']
        peca.estante = request.form['estante']
        peca.prateleira = request.form['prateleira']
        peca.segmento = request.form['segmento']
        imagem = request.files['imagem']
        if imagem:
            imagem_filename = secure_filename(imagem.filename)
            imagem.save(os.path.join(app.config['UPLOAD_FOLDER'], imagem_filename))
            peca.imagem = imagem_filename
        db.session.commit()
        return redirect(url_for('index'))
    return render_template('edit.html', peca=peca)

@app.route('/delete/<int:id>', methods=['POST'])
@login_required
def delete(id):
    peca = Peca.query.get(id)
    if peca:
        if peca.imagem:
            imagem_path = os.path.join(app.config['UPLOAD_FOLDER'], peca.imagem)
            if os.path.exists(imagem_path):  # Verifica se a imagem existe antes de excluir
                os.remove(imagem_path)

        db.session.delete(peca)
        db.session.commit()
    
    return redirect(url_for('index'))

def verificar_e_criar_banco():
    try:
        conexao = pymysql.connect(host="db", user="user", password="password")
        cursor = conexao.cursor()
        
        # Criar o banco de dados
        cursor.execute("CREATE DATABASE IF NOT EXISTS inventario;")
        cursor.execute("USE inventario;")

        # Criar tabelas se não existirem
        cursor.execute("""
        CREATE TABLE IF NOT EXISTS user (
            id INT AUTO_INCREMENT PRIMARY KEY,
            username VARCHAR(50) NOT NULL UNIQUE,
            password VARCHAR(255) NOT NULL,
            criado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        """)

        cursor.execute("""
        CREATE TABLE IF NOT EXISTS peca (
            id INT AUTO_INCREMENT PRIMARY KEY,
            nome VARCHAR(100) NOT NULL,
            descricao TEXT,
            estante VARCHAR(50) NOT NULL,
            prateleira VARCHAR(50) NOT NULL,
            segmento VARCHAR(50) NOT NULL,
            caminho_imagem VARCHAR(255),
            criado_em TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        """)

        conexao.commit()
        conexao.close()
    except Exception as e:
        print(f"Erro ao verificar/criar o banco de dados: {e}")

with app.app_context():
    verificar_e_criar_banco()
    db.create_all()

if __name__ == '__main__':
    db.create_all()
    app.run(host='0.0.0.0',port=8383,debug=True)
