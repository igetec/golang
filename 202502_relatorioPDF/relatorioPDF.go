package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/text/language"
	"golang.org/x/text/message" // Para formatar em R$

	"github.com/jmoiron/sqlx"
	"github.com/jung-kurt/gofpdf"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

type ClientePedido struct {
	NomeCliente string  `db:"nome_cliente"`
	TotalPedido float64 `db:"total_pedido"`
}

func main() {
	// Conexão com o Banco de dados
	db, err := sqlx.Connect("postgres", "user=postgres dbname=igetec sslmode=disable password=postdba")
	if err != nil {
		log.Fatalln(err)
	}

	e := echo.New()

	e.GET("/relvendas", func(c echo.Context) error {
		//-- Query traz o total de vendas por cliente
		query := `
		SELECT c.nome_cliente, COALESCE(SUM(p.total_pedido), 0) as total_pedido
		FROM cliente c
		 left JOIN pedido p ON c.codigo_cliente = p.codigo_cliente
		GROUP BY c.nome_cliente
		ORDER BY total_pedido DESC;		`

		var results []ClientePedido
		err := db.Select(&results, query)
		if err != nil {
			fmt.Println("Falha na Query ao BD:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Falha na Query ao BD:"})
		}

		if len(results) == 0 {
			return c.JSON(http.StatusOK, map[string]string{"message": "Dados não encontrado"})
		}

		//-- Calcula o nome do total de vendas
		var totalSum float64
		for _, row := range results {
			totalSum += row.TotalPedido
		}

		//-- Cria um PDF
		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()

		//-- Adiciona fonte compatível com UTF-8
		//-- https://github.com/owncloud/docs/tree/master/fonts
		pdf.AddUTF8Font("DejaVuSans", "", "DejaVuSans.ttf")
		pdf.AddUTF8Font("DejaVuSans", "B", "DejavuSans-bold.ttf") // Bold
		pdf.SetFont("DejaVuSans", "", 16)

		//-- Título
		title := "Relatório de Vendas por Cliente"
		titleWidth := pdf.GetStringWidth(title)
		pageWidth, _ := pdf.GetPageSize()

		//-- Centraliza o título
		pdf.SetX((pageWidth - titleWidth) / 2)
		pdf.Cell(0, 10, title)
		pdf.Ln(10) // Pula para linha debaixo, dá 10 espaços.

		//-- Desenha uma linha abaixo do título
		dashY := pdf.GetY()
		pdf.Line(10, dashY, pageWidth-10, dashY)
		pdf.Ln(10)

		pdf.SetFont("DejaVuSans", "B", 12) // Seta a fonte

		// Cabaçalhos
		pdf.CellFormat(80, 0, "Cliente", "0", 0, "L", false, 0, "")
		pdf.CellFormat(90, 0, "Total de Vendas", "0", 0, "R", false, 0, "")
		pdf.Ln(5)

		pdf.SetFont("DejaVuSans", "", 12)

		var isGray bool

		for _, row := range results {

			//-- Cria linhas zebradas
			if isGray {
				pdf.SetFillColor(230, 230, 230)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			isGray = !isGray
			pdf.Rect(10, pdf.GetY(), 190, 6, "F")

			pdf.SetTextColor(0, 0, 0)
			pdf.CellFormat(80, 6, row.NomeCliente, "0", 0, "L", false, 0, "")

			p := message.NewPrinter(language.BrazilianPortuguese)
			formattedValue := p.Sprintf("%.2f", row.TotalPedido)
			pdf.CellFormat(90, 6, formattedValue, "0", 1, "R", false, 0, "")
		}

		pdf.Ln(10)
		pdf.Line(10, pdf.GetY(), pageWidth-10, pdf.GetY())
		pdf.Ln(10)

		pdf.SetFont("DejaVuSans", "B", 12)
		pdf.CellFormat(80, 10, "Total Geral", "0", 0, "L", false, 0, "")

		p := message.NewPrinter(language.BrazilianPortuguese)
		formattedValue := p.Sprintf("R$ %.2f", totalSum)
		pdf.CellFormat(90, 10, formattedValue, "0", 1, "R", false, 0, "")

		//-- Cria buffer para o arquivo PDF
		var pdfBuffer bytes.Buffer
		err = pdf.Output(&pdfBuffer)
		if err != nil {
			fmt.Println("falha ao gerar PDF:", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Falha ao gerar PDF"})
		}

		//-- Envia o relatório para download
		return c.Blob(http.StatusOK, "application/pdf", pdfBuffer.Bytes())
	})

	//-- Inicia o servidor
	e.Logger.Fatal(e.Start(":8080"))
}
