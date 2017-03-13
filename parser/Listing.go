package parser

import "fmt"

type Listing struct {
	Title       string
	Description string
	Author      string
	Price       string
}

func (listing *Listing) String() string {
	return listing.Title
}

func (listing *Listing) Print() {
	fmt.Println("Title:", listing.Title)
	fmt.Println("Description:", shortDescription(listing.Description))
	fmt.Println("Author:", listing.Author)
	fmt.Println("Price:", listing.Price)
}
