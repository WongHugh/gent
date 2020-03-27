//Time    : 2020-03-27 16:18
//Author  : Hugh
//File    : CheckCrc.go
//Descripe:

package codec


type NewDataToCheck []byte

type ICheck interface {
	AddCheckData([]byte) (byte,error)
	CheckData(data []byte) bool
}


//校验成功返回True,失败返回False
func (data NewDataToCheck)CheckData() bool {
	sum := make([]byte, 1)
	data_len := len(data)
	if data_len != 0 {
		for i := 0; i < data_len-1; i++ {
			sum[0] += data[i]
		}
		if sum[0] == data[data_len-1] {
			return true
		}
	}
	return false
}

//ADD check sum data after byte data
func (data NewDataToCheck)AddCheckSum() []byte {
	rec_len := len(data)
	sum := make([]byte, 1)
	for i := 0; i < rec_len; i++ {
		sum[0] += data[i]
	}
	data = append(data, sum[0])
	return data
}