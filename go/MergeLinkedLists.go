package main

import "fmt"

/**
 * Definition for singly-linked list.
 * type ListNode struct {
 *     Val int
 *     Next *ListNode
 * }
 */

type ListNode struct {
	Val  int
	Next *ListNode
}

func mergeTwoLists(list1 *ListNode, list2 *ListNode) *ListNode {
	final := &ListNode{}

	// check if lists are empty
	if list1 == nil || list2 == nil {
		if list1 != nil {
			return list1
		} else if list2 != nil {
			return list2
		}
		return nil
	}

	// if lists are not empty:

	return final
}

func main() {
	list1 := &ListNode{
		Val: 1,
		Next: &ListNode{
			Val: 2,
			Next: &ListNode{
				Val:  3,
				Next: &ListNode{},
			},
		},
	}
	list2 := &ListNode{
		Val: 1,
		Next: &ListNode{
			Val: 3,
			Next: &ListNode{
				Val:  5,
				Next: &ListNode{},
			},
		},
	}

	fmt.Println(list1.Val, list2.Next.Next.Next)
}
