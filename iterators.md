## Pull iterators

Needs a struct where the current state is kept (f.ex the index)
This struct has a next() method that returns the next item.
When there are no more items, the next() function returns err=DONE,
and we use standard error checking to exit the loop.

This is typically done with an `if r.index>r.end ? fail(DONE)` instruction

This implementation avoids any problems with closures or stack frames.

For this to work, we need a state data structure with a method called 
next() on that data. There is also typically a function that creates
a new state structure.

## Simple iterator over integers

    type RangeIterator struct {
        start I32
        end I32
    }
    
    func range(start I32, end I32) RangeIterator {
        r = new(RangeIterator)
        r.start = start
        r.end = end
        return r
    }

    next(r RangeIterator) I32 {
        if r.start>r.end ? fail(DONE)
        return r.start++   // Post increment
    }

### Simple excplicite use

    func main() {
        r = range(1, 10)
        for {
            i = next(r)
            print("%d", i)
        }
    }

With syntactic sugar, where we assume the next() function exists,
and has just that name. The iterator state is held in an implicit
variable called `state`, and state.next() is called for each loop.
We assume the next(state) function exists.

    func main() {
        for i=range(1,10) {
            print("%d", i)
        }
    }

## Bidirectional range iterator

If start<end it iterates from start up to end inclusive
If start>end it iterates from start down up to end inclusive
If start==end it executes the body once with that value

    type RangeIterator struct {
        done Bool
        start I32
        end I32
    }
    
    func range(start I32, end I32) func() I32 {
        r = new(RangeIterator)
        r.start = start
        r.end = end
        return r
    }

    next(r RangeIterator) I32 {
        if r.done ? fail(DONE)
        value = r.start
        if r.start<r.end {
            r.start++
            done = r.start>r.end
        } else {
            r.start--
            done = r.start<r.end
        }
    }

    func main() {
        r = range(1, 10)
        for {
            i = next(r)
            print("%d", i)
        }
    }

## Additional functions

range() written with explicite call to new()

    func range(start I32, end I32) RangeIterator {
        r = new(RangeIterator)
        r.index = start
        r.end = end
        return r
    }

We can reuse an iteator:

    restart(r RangeIterator, start I32, end I32) {
        r.index = start
        r.end = end
    ]


## Iterator over a tree

This is a bit more complicated, because the state has to keep
track of which children has been visited.
The order of visits is determined by the next() function.

    type Node = struct {
        left Node
        right Node
        data String
    }

    type State struct {
        left_visited Bool
        right_visited Bool
        self_visited Bool
        node Any
    }

    type TreeIterator struct {
        parent TreeIterator
        left_visited Bool
        right_visited Bool
        self_visited Bool
        node Any
    }

    func traverse(node Node, parent TreeIterator) TreeIterator {
        r=new(TreeIterator)
        r.node = node
        r.parent = parent
        // not needed: r.left_visited = false
        // not needed: r.right_visited = false
        // not needed: r.self_visited = false
        return r
    }

    func next(r TreeIterator) Node { 
        for {
            if !r.left_visited  {
                r.left_visited = true
                if r.node.left!=nil {
                    r = traverse(r.node.left, r)
                } 
            } else if !r.right_visited {
                r.right_visited = true
                if  r.node.right!=nil {
                    r = traverse(r.node.left, r)
                }
            } else !r.self_visited{
                r.self_visited = true
                return r.node
            } else {  
                // Exit loop if this is the root node
                if r.parent==nil ? fail(DONE)
                // all three flags are true so we return back to parent
                r = r.parent
            }
        }
    }

### Tree iterator usage with explicit iterator

    func main() {
        tree = ....
        iter = traverse(tree, nil)
        for  {
            node = next(iter) else break
            printf("%s\n", node.name)
        }
    }

### Tree iterator with syntactic sugar

    func main() {
        tree = ....
        for n=traverse(tree).next() {
            printf("%s\n", n.name)
        }
    }

## Iterator over slice element

    type IntSliceIterator struct {
        index I32
        data  []I32
    }

    iterate(a []I32) IntSliceIterator {
        it =new(IntSliceIterator)
        it.data = a
        return it
    }

    next(s IntSliceIterator) any {
        if r.index>r ? fail(DONE)
        return s[r.index++]         // NB: Post-increment
    }

Do we have to keep data in the iterator? Yes, but we only need 
a pointer to the original data. The pointer is a temporary pointer 
that does not own the buffer itself.

### Usage of slice iterator with explicit iterator 

    func main() {
        Seq = []I32{1,2,3,4,5}
        iter = iterate(Seq)
        for {   
            e = iter.next()
            print("%d", e))
        }
    }

### Usage of slice iterator with syntactic sugar:

    func main() {
        Seq = []I32{1,2,3,4,5}
        for e = iterate(Seq).next() {   
            e = iter.next()
            print("%d", e))
        }
    }
 

### Usage iterating over two slices simultaneously

    func main() {
        Seq1 = []I32{1,2,3,4,5}s
        Seq2 = []I32{1,2,3,4,5}
        iter1 = iterate(Seq1)
        iter2 = iterate(Seq2)
        for {   
            e1 = iter1.next()
            e2 = iter2.next()
            print("%d %d", e1, e2))
        }
    }

### Filtering

    next_pos(s IntSliceIterator) I32 {
        for {
            e = next(s) else fail(DONE)
            if e>=0 return e
        }
    }

    func main() {
        Seq = []I32{1,2,3,4,5}
        for e = next_pos(iterate(Seq)) {
            print("%d", e))
        }
    }

## Chaining

   e = iter.next_if(func(x I32) bool {return x>0})


## Split data into lines

    type LineIterator struct {
        data String
        pos I32
    }
    
    func next(s LineIterator) String {
        if s.pos>=len(s.data) fail(DONE)
        start = s.pos
        end = s.pos+1
        loop {
            if s.data[end] == "\n" || s.pos>=len(s.data) {
                s.pos = end+1
                return s.data[start, end]  // Post increment
            }
            end++
        }
    }

    func Lines(data []byte, yield func([]byte)) {
        for len(data)>0 {
            line, data =Cut(data,'n\')
            yield(line)
        }
    }

    main() {
        data = ...
        it = line_iterator(data) 
        for line=Lines(data) {
            handle_line(line)
        }
