# Disko 💿

Analisador de uso de disco no terminal, inspirado no WinDirStat. Escrito em Go, com interface TUI construída com Bubble Tea.

## Sobre

Disko escaneia um diretório recursivamente e permite navegar pela árvore de arquivos direto no terminal, ordenada por tamanho, para identificar rapidamente o que está consumindo espaço em disco.

## Funcionalidades
1. Escaneamento recursivo com scan paralelizado via worker pool
2. Navegação interativa por diretórios (entrar/voltar) com histórico de posição do cursor
3. Ordenação automática por tamanho (maiores primeiro), em todos os níveis da árvore
4. Feedback em tempo real durante o scan (contador de arquivos + caminho atual sendo lido)
5. Formatação legível de tamanhos (B, KB, MB, GB, TB, PB)
6. Exclusão automática de /proc, /sys e /dev ao escanear a partir da raiz /

## Arquitetura

  main.go
  
  internal/
  
  scanner/
  
  tree/

O scan roda sobre um pool fixo de 128 goroutines consumindo de um canal de jobs. Cada job representa um diretório a ser lido, ao encontrar subdiretórios, o worker enfileira um novo job para eles.

### Árvore

Estrutura simples: Entry guarda nome, tipo e filhos. CalcSizes percorre a árvore recursivamente somando o tamanho dos filhos para determinar o tamanho de cada pasta.

## Como rodar

git clone https://github.com/leomorae-s/Disko.git

cd Disko

go run main.go [caminho]

Se nenhum caminho for informado, o scan parte do diretório home do usuário.

## Controles

↑ / k	mover cursor para cima

↓ / j	mover cursor para baixo

Enter / → / l	entrar na pasta selecionada

Esc / ← / h / Backspace	voltar para a pasta anterior

q / Ctrl+C	sair

## Status

MVP funcional: scan concorrente, navegação completa e ordenação por tamanho já implementados.

## Ideias futuras que podem ou não ser implementadas:

1. GUI usando Fyne
2. Implementação do algoritmo squarified treemap para representação visual do tamanho dos diretórios

Esse projeto foi criado por curiosidade a respeito do WinDirStats. A implementação neste projeto foi simplificada, a representação visual é feita via lista ordenada por tamanho. A visualização em treemap é uma evolução possível, mas fora do escopo do MVP atual.
