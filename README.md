# SafeBox

O **SafeBox** é uma solução robusta de armazenamento em nuvem projetada para oferecer segurança, escalabilidade e desempenho excepcionais.

## Tecnologias Utilizadas

- **Go**: Linguagem de programação principal para o desenvolvimento do backend.
- **Echo**: Framework web em Go para construção de APIs rápidas e eficientes.
- **GORM**: ORM (Object Relational Mapping) para Go, facilitando a interação com bancos de dados relacionais.
- **Logrus**: Biblioteca de logging estruturado para Go, proporcionando logs detalhados e configuráveis.
- **JWT (JSON Web Tokens)**: Implementação de autenticação e autorização segura.
- **OAuth**: Protocolo de autorização para integração com serviços de terceiros.
- **Cloudflare R2**: Solução de armazenamento de objetos compatível com S3, utilizada para armazenar arquivos de forma eficiente e econômica.
- **Prometheus**: Sistema de monitoramento e alerta para coletar métricas detalhadas.
- **Grafana**: Plataforma de visualização de dados para criação de dashboards interativos e monitoramento em tempo real.

## Funcionalidades

- **Armazenamento Seguro**: Upload e download de arquivos com criptografia em trânsito e em repouso.
- **Controle de Acesso**: Autenticação via JWT e autorização baseada em OAuth para integração com serviços externos.
- **Monitoramento Avançado**: Coleta de métricas detalhadas utilizando Prometheus, com visualização em dashboards Grafana.
- **Escalabilidade**: Utilização de AWS e Cloudflare R2 para garantir alta disponibilidade e escalabilidade do sistema.

## Instalação

1. Clone o repositório:
   ```bash
   git clone https://github.com/FelipePn10/SafeBox.git
   ```
2. Navegue até o diretório do projeto:
   ```bash
   cd SafeBox
   ```
3. Instale as dependências:
   ```bash
   go mod tidy
   ```
4. Configure as variáveis de ambiente necessárias, como credenciais da AWS, configurações do banco de dados e parâmetros de autenticação.
5. Inicie o servidor:
   ```bash
   go run main.go
   ```

## Contribuição

Contribuições são bem-vindas!
Para contribuir, siga os seguintes passos:
1. Fork o repositório.
2. Crie uma branch para sua feature (`git checkout -b feature/nome-da-feature`).
3. Commit suas alterações (`git commit -am 'Adiciona nova feature'`).
4. Push para a branch (`git push origin feature/nome-da-feature`).
5. Abra um Pull Request detalhando as alterações realizadas.

## Licença

Este projeto está licenciado sob a MIT License - veja o arquivo [LICENSE](LICENSE) para mais detalhes.

## Contato

Para dúvidas ou mais informações, entre em contato através do e-mail: felipe.pn10@gmail.com

Para mais detalhes sobre o projeto, visite o repositório no GitHub: [https://github.com/FelipePn10/SafeBox](https://github.com/FelipePn10/SafeBox) 
