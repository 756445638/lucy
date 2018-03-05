import src.cmd.lucy.command as command


import os

javacode = '''





package lucy.lang;
public class ArrayTTT   {
	public int start;
	public int end; // not include
	public int cap;
	static String outOfRagneMsg = "index out range";
	public TTT[] elements;
	public int size(){
		return this.end - this.start;
	}
	public int start(){
        return this.start;
	}
	public int end(){
         return this.end;
	}
	public int cap(){
         return this.end;
	}
	public ArrayTTT(TTT[] values,int end){
		this.start = 0;
		this.end = end;
		this.cap = values.length;
		this.elements = values;
	}
	private ArrayTTT(){

	}
	public ArrayTTT slice(int start,int end){
		if(end  < 0 ){
		      end = this.end - this.start;  // whole length
		}
		ArrayTTT result = new ArrayTTT();
		if(start < 0 || start > end || end + this.start > this.end){
			throw new ArrayIndexOutOfBoundsException(outOfRagneMsg);
		}
		result.elements = this.elements;
		result.start = this.start + start;
		result.end = this.start + end;
		result.cap = this.cap;
		return result;
	}
	public TTT get(int index){
		if(this.start + index >= this.end || index < 0){
			throw new ArrayIndexOutOfBoundsException(outOfRagneMsg);
		}
		return this.elements[this.start + index];
	}
	public void set(int index,TTT v){
		if(this.start + index >= this.end || index < 0){
			new ArrayIndexOutOfBoundsException(outOfRagneMsg);
		}
		this.elements[this.start + index] = v;
	}
	public ArrayTTT append(TTT e){
		if(this.end < this.cap){
		}else{
			this.expand(this.cap * 2);
		}
		this.elements[this.end++] = e;
		return this;
	}
	private void expand(int cap){
		if(cap <= 0){
		    cap = 10;
		}
		TTT[] eles = new TTT[cap];
		int length = this.size();
		for(int i = 0;i < length;i++){
			eles[i] = this.elements[i + this.start];
		}
		this.start = 0;
		this.end = length;
		this.cap = cap;
		this.elements = eles;
	}
	public ArrayTTT append(TTT[] es){
		if(this.end + es.length < this.cap){
		}else {
			this.expand((this.cap + es.length) * 2);
		}
		for(int i = 0;i < es.length;i++){
			this.elements[this.end + i] = es[i];
		}
		this.end += es.length;
		return this;
	}
	public String toString(){
	    String s = "[";
	    int size = this.end - this.start;
	    for(int i= 0;i < size;i ++){
            s += this.elements[this.start + i ];
            if(i != size -1){
                s += " ";
            }
	    }
	    s += "]";
	    return s;
	}

}













'''







class MkLucyArray(command.Command):
    def __init__(self):
        pass

    def runCommand(self,args):
        javaBasicTypes = ["boolean","byte","char","short","int","float","long","double","Object","String"]
        files = []
        for t in javaBasicTypes:
            s = javacode.replace("TTT",t)
            name = "Array" + t
            s = s.replace("ArrayTTT",name)
            f = open(name + ".java",  'w')
            files.append(name + ".java")
            f.write(s)
        for f in files:
            os.system("javac ./" + f  )








