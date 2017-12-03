import src.cmd.lucy.command as command
import sys
import os
from optparse import OptionParser
import struct
import  json







class Declass(command.Command):
    def __init__(self):
        self.__src = ""
        self.__dest = ""
        self.__help_msg = "declass jvm class files,command line args are -src and -dest"

    def __parseParameter(self,args):
        parser = OptionParser(usage=self.__help_msg)
        parser.add_option("--src",action="store",type="string",dest="src",help="source directory")
        parser.add_option("--dest", action="store", type="string", dest="dest", help="destination directory")
        opt,args = parser.parse_args(args)
        if opt.dest == "" or opt.src == "":
            Declass.static_usage()
            sys.exit(1)
        self.__src = opt.src
        self.__dest = opt.dest
        return 0

    def __parse(self):
        if os.path.exists(self.__src) == False:
            print("src %s directory is not exits" % (self.__src))
            return
        if os.path.exists(self.__dest) == False:
            os.mkdir(self.__dest)

        self.__parseDir(self.__src,self.__dest)

        return 0

    def __parseDir(self,src ,dest):
        print("read dir " + src)
        if os.path.isdir(src)  == False :
            return
        fis = os.listdir(src)
        for d in fis:
            if d.endswith(".class"):  # class file
                if d.find("$") != -1:  #name contains $ means a inner class
                    continue
                self.__parseFile("%s/%s" % (src,d),dest,d)
            else:
                self.__parseDir("%s/%s" % (src,d),"%s/%s" % (dest,d))

    def __parseFile(self,src,dest,filename):
        p = JvmClassParser(src,dest)
        ret = p.parse()
        if "ok" not in ret:
            print("declass file %s failed,err:%s" % (src,ret.reason))
            return
        ret = ret["class"].output()
        if os.path.exists(dest) == False:
            os.mkdir(dest)

        filename = "%s/%s.json" % (dest,filename.rstrip(".class"))
        fd = open(filename,'w')
        fd.write(ret)


    def static_usage():
        print("declass jvm class files,command line args are -src and -dest")

    def runCommand(self,args):
        args = args[1:] # skip run command
        if self.__parseParameter(args) != 0:
            sys.exit(1)

        if 0 != self.__parse():
            sys.exit(2)




class JvmClass:
    def __init__(self):
        self.magic = 0
        self.minorVersion = 0
        self.majorVersion = 0
        self.constPool = []
        self.access_flags = 0
        self.this_class = 0
        self.super_class = 0
        self.interfaces = []  # interface counts
        self.fields = []
        self.methods = []
        self.attrs = []
        pass
    def output(self):
        output = {}
        output["magic"] = self.magic
        output["minorVersion"] = self.minorVersion
        output["majorVersion"] = self.majorVersion
        output["access_flags"] = self.access_flags
        output["this_class"] = self.constPool[self.constPool[self.this_class]["name_index"]]["string"]
        output["super_class"] = self.constPool[self.constPool[self.super_class]["name_index"]]["string"]
        output["fields"] = self.__mk_fileds()
        output["methods"] = self.__mk_methods()
        x = json.JSONEncoder()
        return x.encode(output)




    def __mk_methods(self):
        ms = []
        for v in self.methods:
            m = {}
            m["access_flags"] = v["access_flags"]
            m["name"] =  self.constPool[v["name_index"]]["string"]
            descriptor = self.constPool[v["descriptor_index"]]["string"]
            m["typ"] = self.__parseMethodDescriptor(descriptor)
            ms.append(m)
        return ms

    def __parseMethodDescriptor(self,d):
        ret = {}
        ret["parameters"] = []
        ret["return"] = ""
        for v in d[1:d.index(')')].split(";"):
            if len(v) == 0 :
                continue
            if v[0] == "L" or v[0] == "[":
                ret["parameters"].append(v)
            else:
                for vv in v:
                    ret["parameters"].append(vv)
        ret["return"] = d[d.index(")") + 1 :]
        if ret["return"][len(ret["return"])-1] == ";": #cut ;
            ret["return"] = ret["return"][0:len(ret["return"])-1]
        return ret


    def __mk_fileds(self):
        fs = []
        for v in self.fields:
            f = {}
            f["access_flags"] = v["access_flags"]
            f["name"] = self.constPool[v["name_index"]]["string"]
            f["descriptor"] = self.constPool[v["descriptor_index"]]["string"]
            fs.append(f)
        return fs


CONSTANT_TAG_Class  = 7
CONSTANT_TAG_Fieldref  = 9
CONSTANT_TAG_Methodref =  10
CONSTANT_TAG_InterfaceMethodref = 11
CONSTANT_TAG_String = 8
CONSTANT_TAG_Integer = 3
CONSTANT_TAG_Float = 4
CONSTANT_TAG_Long = 5
CONSTANT_TAG_Double = 6
CONSTANT_TAG_NameAndType = 12
CONSTANT_TAG_Utf8 = 1
CONSTANT_TAG_MethodHandle = 15
CONSTANT_TAG_MethodType = 16
CONSTANT_TAG_InvokeDynamic = 18







class JvmClassParser:
    def __init__(self,filepath,destfilepath):
        self.__filepath = filepath
        self.__descfilepath = destfilepath
        self.__result = JvmClass() # hold result in this

    def parse(self):  # file is definitely exits
        fd = open(self.__filepath,"rb")
        try:
            self.__content = fd.read()
        finally:
            fd.close()
        #magic and version
        ok = self.__parseMagicAndVersion()
        if 0 != ok:
            return {"reason": ok}
        # const pool
        ok = self.__parseConstPool()
        if 0 != ok:
            return {"reason": ok}
        #access and interfaces
        ok = self.__parseInterfaces()
        if 0 != ok:
            return {"reason": ok}
        # fields
        ok = self.__parseFileds()
        if 0 != ok:
            return {"reason": ok}
        #methods
        ok = self.__parseMethods()
        if 0 != ok:
            return {"reason": ok}
        self.__result.attrs = self.__parseAttibute()
        return {"ok":True,"class":self.__result}


    def __parseInterfaces(self):
        ret = struct.unpack_from("!HHHH",self.__content)
        self.__result.access_flags = ret[0]
        self.__result.this_class = ret[1]
        self.__result.super_class = ret[2]
        self.__result.interfaces  = [] # interface counts
        self.__content = self.__content[8:]
        if 0 == ret[3]:
            return 0
        for i in range(0,ret[3]):
            ret = struct.unpack_from("!H",self.__content)
            self.__result.interfaces.append({"index":ret[0]})
            self.__content = self.__content[2:]
        return 0

    def __parseFileds(self):
        ret = struct.unpack_from("!H",self.__content)
        self.__content = self.__content[2:]
        self.__result.fields = []
        for i in range(0,ret[0]):
            ret = struct.unpack_from("!HHH",self.__content)
            self.__content = self.__content[6:]
            attrs = self.__parseAttibute()
            self.__result.fields.append({"access_flags": ret[0],"name_index": ret[1],"descriptor_index": ret[2],"attributes": attrs})
        return 0

    def __parseAttibute(self):
        ret = struct.unpack_from("!H",self.__content)
        self.__content = self.__content[2:]
        attrs = []
        for i in range(0,ret[0]):
            ret = struct.unpack_from("!HI",self.__content)
            length = ret[1]
            self.__content = self.__content[6:]
            attrs.append({"name_index":ret[0],"length":length,"bytes":self.__content[0:length]})
            self.__content = self.__content[length:]
        return attrs

    def __parseMethods(self):
        ret = struct.unpack_from("!H", self.__content)
        self.__content = self.__content[2:]
        self.__result.methods = []
        for i in range(0, ret[0]):
            ret = struct.unpack_from("!HHH", self.__content)
            self.__content = self.__content[6:]
            attrs = self.__parseAttibute()
            self.__result.methods.append({"access_flags": ret[0], "name_index": ret[1], "descriptor_index": ret[2], "attributes": attrs})
        return 0

    def __parseConstPool(self):
        ret = struct.unpack_from("!H",self.__content[0:])
        size = ret[0]
        self.__result.constPool = [{}]
        self.__content = self.__content[2:]
        print(self.__filepath)
        i = 1
        while True:
            if i > size -1:
                break
            ret = struct.unpack_from("!B",self.__content)
            tag = ret[0]
            self.__content = self.__content[1:]  # skip tag
            if tag == CONSTANT_TAG_Class:
                ret = struct.unpack_from("!H",self.__content)
                self.__content = self.__content[2:]
                self.__result.constPool.append({"tag":tag,"name_index": ret[0]})
                i += 1
                continue
            if tag == CONSTANT_TAG_Fieldref:
                ret = struct.unpack_from("!HH",self.__content)
                self.__content = self.__content[4:]
                self.__result.constPool.append({"tag": tag, "class_index": ret[0],"name_and_type_index": ret[1]})
                i += 1
                continue
            if tag == CONSTANT_TAG_Methodref:
                ret = struct.unpack_from("!HH", self.__content)
                self.__content = self.__content[4:]
                self.__result.constPool.append({"tag": tag, "class_index": ret[0], "name_and_type_index": ret[1]})
                i += 1
                continue
            if tag == CONSTANT_TAG_InterfaceMethodref:
                ret = struct.unpack_from("!HH", self.__content)
                self.__content = self.__content[4:]
                self.__result.constPool.append({"tag": tag, "class_index": ret[0], "name_and_type_index": ret[1]})
                i += 1
                continue
            if tag == CONSTANT_TAG_String:
                ret = struct.unpack_from("!H", self.__content)
                self.__content = self.__content[2:]
                self.__result.constPool.append({"tag": tag, "string_index": ret[0]})
                i += 1
                continue
            if tag == CONSTANT_TAG_Integer:
                self.__result.constPool.append({"tag": tag, "bytes": self.__content[0:4]})
                self.__content = self.__content[4:]
                i += 1
                continue
            if CONSTANT_TAG_Float == tag:
                self.__result.constPool.append({"tag": tag, "bytes": self.__content[0:4]})
                self.__content = self.__content[4:]
                i += 1
                continue
            if CONSTANT_TAG_Long == tag:
                self.__result.constPool.append({"tag": tag, "hight_bytes": self.__content[0:4],"low_bytes": self.__content[4:8]}) # n
                self.__result.constPool.append({})  # n+1 not available
                i += 2
                self.__content = self.__content[8:]
                continue
            if CONSTANT_TAG_Double == tag:
                self.__result.constPool.append({"tag": tag, "hight_bytes": self.__content[0:4], "low_bytes": self.__content[4:8]})
                self.__result.constPool.append({})
                i += 2
                self.__content = self.__content[8:]
                continue
            if CONSTANT_TAG_NameAndType == tag:
                ret = struct.unpack_from("!HH", self.__content)
                self.__content = self.__content[4:]
                self.__result.constPool.append({"tag": tag, "name_index": ret[0], "descriptor_index": ret[1]})
                i += 1
                continue
            if CONSTANT_TAG_Utf8 == tag:
                ret = struct.unpack_from("!H", self.__content)
                self.__content = self.__content[2:]
                length = ret[0]
                self.__result.constPool.append({"tag": tag, "length":length, "string": self.__content[0:length].decode()})
                self.__content = self.__content[length:]
                i += 1
                continue
            if CONSTANT_TAG_MethodHandle == tag:
                ret = struct.unpack_from("!BH", self.__content)
                self.__content = self.__content[3:]
                self.__result.constPool.append({"tag": tag, "reference_kind": ret[0], "reference_index": ret[1]})
                i += 1
                continue
            if CONSTANT_TAG_MethodType == tag:
                ret = struct.unpack_from("!H", self.__content)
                self.__content = self.__content[2:]
                self.__result.constPool.append({"tag": tag, "descriptor_index": ret[0]})
                i += 1
                continue
            if CONSTANT_TAG_InvokeDynamic == tag:
                ret = struct.unpack_from("!HH", self.__content)
                self.__content = self.__content[4:]
                self.__result.constPool.append({"tag": tag, "bootstrap_method_attr_index": ret[0], "name_and_type_index": ret[1]})
                i += 1
                continue
            return "un know tag: %d" % (tag)

        return 0


    def __parseMagicAndVersion(self):
        ret = struct.unpack_from("!I",self.__content)
        self.__result.magic = ret[0]
        self.__content = self.__content[4:]
        ret = struct.unpack_from("!HH",self.__content)
        self.__result.minorVersion = ret[0]
        self.__result.majorVersion = ret[1]
        self.__content = self.__content[4:]
        return 0



