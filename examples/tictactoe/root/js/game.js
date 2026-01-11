// Tic-Tac-Toe Game Client
// Powered by DBasic

class TicTacToe {
    constructor() {
        this.board = ['', '', '', '', '', '', '', '', ''];
        this.playerSymbol = 'X';
        this.aiSymbol = 'O';
        this.gameOver = false;
        this.wins = 0;
        this.losses = 0;
        this.draws = 0;

        this.initElements();
        this.bindEvents();
        this.newGame();
    }

    initElements() {
        this.cells = document.querySelectorAll('.cell');
        this.messageEl = document.getElementById('message');
        this.symbolsEl = document.getElementById('symbols');
        this.newGameBtn = document.getElementById('newGameBtn');
        this.resetRecordBtn = document.getElementById('resetRecordBtn');
        this.winsEl = document.getElementById('wins');
        this.lossesEl = document.getElementById('losses');
        this.drawsEl = document.getElementById('draws');
        this.boardEl = document.getElementById('board');
    }

    bindEvents() {
        this.cells.forEach((cell, index) => {
            cell.addEventListener('click', () => this.handleCellClick(index));
        });

        this.newGameBtn.addEventListener('click', () => this.newGame());
        this.resetRecordBtn.addEventListener('click', () => this.resetRecord());
    }

    async newGame() {
        this.setLoading(true);

        try {
            const response = await fetch('/api/newgame');
            const data = await response.json();
            this.updateFromResponse(data);
        } catch (error) {
            console.error('Error starting new game:', error);
            this.messageEl.textContent = 'Error connecting to server. Please refresh.';
        }

        this.setLoading(false);
    }

    async handleCellClick(position) {
        // Ignore if game is over or cell is taken
        if (this.gameOver || this.board[position] !== '') {
            return;
        }

        this.setLoading(true);

        try {
            const boardStr = this.board.join(',');
            const url = `/api/move?board=${encodeURIComponent(boardStr)}&player=${this.playerSymbol}&ai=${this.aiSymbol}`;

            const response = await fetch(url, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ Position: position }),
            });

            const data = await response.json();
            this.updateFromResponse(data);
        } catch (error) {
            console.error('Error making move:', error);
            this.messageEl.textContent = 'Error connecting to server.';
        }

        this.setLoading(false);
    }

    async resetRecord() {
        this.setLoading(true);

        try {
            const response = await fetch('/api/reset');
            const data = await response.json();
            this.wins = data.Wins;
            this.losses = data.Losses;
            this.draws = data.Draws;
            this.updateStats();
            this.messageEl.textContent = data.Message || 'Record reset!';
        } catch (error) {
            console.error('Error resetting record:', error);
        }

        this.setLoading(false);
    }

    updateFromResponse(data) {
        // DBasic generates PascalCase JSON field names
        this.board = data.Board;
        this.playerSymbol = data.PlayerSymbol;
        this.aiSymbol = data.AISymbol;
        this.gameOver = data.GameOver;
        this.wins = data.Wins;
        this.losses = data.Losses;
        this.draws = data.Draws;

        this.messageEl.textContent = data.Message;
        this.symbolsEl.textContent = `You are playing as ${this.playerSymbol}`;

        this.renderBoard();
        this.updateStats();

        // Highlight winning cells if game is over
        if (this.gameOver && data.Winner) {
            this.highlightWinner(data.Winner);
        }
    }

    renderBoard() {
        this.cells.forEach((cell, index) => {
            const value = this.board[index];
            cell.textContent = value;
            cell.className = 'cell';

            if (value === 'X') {
                cell.classList.add('x', 'taken');
            } else if (value === 'O') {
                cell.classList.add('o', 'taken');
            }

            if (this.gameOver) {
                cell.classList.add('game-over');
            }
        });
    }

    highlightWinner(winner) {
        const winPatterns = [
            [0, 1, 2], [3, 4, 5], [6, 7, 8], // rows
            [0, 3, 6], [1, 4, 7], [2, 5, 8], // columns
            [0, 4, 8], [2, 4, 6]  // diagonals
        ];

        for (const pattern of winPatterns) {
            const [a, b, c] = pattern;
            if (this.board[a] === winner &&
                this.board[b] === winner &&
                this.board[c] === winner) {
                this.cells[a].classList.add('winner');
                this.cells[b].classList.add('winner');
                this.cells[c].classList.add('winner');
                break;
            }
        }
    }

    updateStats() {
        this.winsEl.textContent = this.wins;
        this.lossesEl.textContent = this.losses;
        this.drawsEl.textContent = this.draws;
    }

    setLoading(loading) {
        if (loading) {
            this.boardEl.classList.add('loading');
        } else {
            this.boardEl.classList.remove('loading');
        }
    }
}

// Initialize game when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new TicTacToe();
});
